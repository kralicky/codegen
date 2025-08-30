package cliutil

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"

	"buf.build/go/protovalidate"
	"buf.build/go/protoyaml"
	"google.golang.org/protobuf/proto"
)

var ErrAborted = errors.New("aborted by user")

var ErrNoEditor = fmt.Errorf("no available editor; please set the EDITOR environment variable and try again")

func EditInteractive[T proto.Message](spec T, comments ...string) (T, error) {
	var err error
	for {
		extraComments := slices.Clone(comments)
		if err != nil {
			extraComments = []string{fmt.Sprintf("error: %v", err)}
		}
		var editedSpec T
		editedSpec, err = tryEdit(spec, extraComments)
		if err != nil {
			if errors.Is(err, ErrAborted) || errors.Is(err, ErrNoEditor) {
				return editedSpec, err
			}
			continue
		}
		return editedSpec, nil
	}
}

var validator, _ = protovalidate.New()

type protovalidateValidator struct {
	protovalidate.Validator
}

func (v protovalidateValidator) Validate(msg proto.Message) error {
	return v.Validator.Validate(msg)
}

func LoadFromFile[T proto.Message](spec T, path string) error {
	var f *os.File
	if path == "-" {
		f = os.Stdin
	} else {
		var err error
		f, err = os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
	}
	bytes, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	if len(bytes) == 0 {
		return fmt.Errorf("file is empty")
	}

	if err := (protoyaml.UnmarshalOptions{
		Path:      path,
		Validator: protovalidateValidator{validator},
	}).Unmarshal(bytes, spec); err != nil {
		return fmt.Errorf("error unmarshalling json: %w", err)
	}
	return nil
}

func tryEdit[T proto.Message](spec T, extraComments []string) (T, error) {
	for i, comment := range extraComments {
		if !strings.HasPrefix(comment, "#") {
			extraComments[i] = "# " + comment
		}
	}

	var nilT T
	inputData, err := protoyaml.Marshal(spec)
	if err != nil {
		return nilT, err
	}

	// Add comments to the JSON
	comments := append([]string{
		"# Edit the configuration below. Comments are ignored.",
		"# If everything is deleted, the operation will be aborted.",
	}, extraComments...)
	specWithComments := strings.Join(append(comments, string(inputData)), "\n")

	// Create a temporary file for editing
	tmpFile, err := os.CreateTemp("", "cli-temp-editing-*.yaml")
	if err != nil {
		return nilT, err
	}
	defer os.Remove(tmpFile.Name())

	// Write the JSON with comments to the temporary file
	if _, err := tmpFile.WriteString(specWithComments); err != nil {
		return nilT, err
	}
	if err := tmpFile.Close(); err != nil {
		return nilT, err
	}

	// Open the temporary file in the user's preferred editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editors := []string{"nvim", "vim", "vi"}
		for _, e := range editors {
			if _, err := exec.LookPath(e); err == nil {
				editor = e
				break
			}
		}
	}

	args := []string{tmpFile.Name()}

	if _, err := exec.LookPath(editor); err != nil {
		return nilT, ErrNoEditor
	}

	cmd := exec.Command(editor, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nilT, fmt.Errorf("editor command failed: %w", err)
	}

	// Read the edited JSON
	editedBytes, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return nilT, err
	}

	editedBytes = bytes.TrimSpace(editedBytes)

	// Remove comments and empty lines
	editedLines := strings.Split(string(editedBytes), "\n")
	filteredLines := make([]string, 0, len(editedLines))
	for _, line := range editedLines {
		trimmedLine := strings.TrimSpace(line)
		if len(trimmedLine) > 0 && !strings.HasPrefix(trimmedLine, "#") {
			filteredLines = append(filteredLines, line)
		}
	}

	// If everything is deleted, abort the operation
	if len(filteredLines) == 0 {
		return nilT, ErrAborted
	}

	editedSpec := spec.ProtoReflect().New()

	unmarshalOpts := protoyaml.UnmarshalOptions{
		Validator: protovalidateValidator{validator},
	}

	if err := unmarshalOpts.Unmarshal([]byte(strings.Join(filteredLines, "\n")), editedSpec.Interface()); err != nil {
		return nilT, err
	}

	return editedSpec.Interface().(T), nil
}
