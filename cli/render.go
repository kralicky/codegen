package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/bufbuild/protoyaml-go"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

type Writer interface {
	Print(i ...any)
	Println(i ...any)
	Printf(format string, i ...any)
	PrintErr(i ...any)
	PrintErrln(i ...any)
	PrintErrf(format string, i ...any)
}

// An optional interface that can be implemented by an RPC response type to
// control how it is rendered to the user.
type TextRenderer interface {
	// RenderText renders the message to the given writer in a human-readable
	// format.
	RenderText(out Writer)
}

func RenderOutput(cmd *cobra.Command, response proto.Message) {
	outputFormat, _ := cmd.Flags().GetString("output")
	outWriter := cmd.OutOrStdout()

	switch outputFormat {
	case "json":
		fmt.Fprintln(outWriter, protojson.MarshalOptions{}.Format(response))
	case "json,multiline":
		fmt.Fprintln(outWriter, protojson.MarshalOptions{
			Multiline: true,
		}.Format(response))
	case "yaml":
		out, err := protoyaml.MarshalOptions{Indent: 2}.Marshal(response)
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		fmt.Fprintln(outWriter, string(out))
	case "text":
		if renderer, ok := response.(TextRenderer); ok {
			renderer.RenderText(cmd)
			return
		}
		fmt.Fprintln(outWriter, prototext.MarshalOptions{
			Multiline: true,
		}.Format(response))
	default:
		cmd.PrintErrln("Unknown output format:", outputFormat)
	}
}

func RenderStreamingOutput[T proto.Message, S interface {
	Recv() (T, error)
	grpc.ClientStream
}](cmd *cobra.Command, stream S) error {
	for {
		msg, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		RenderOutput(cmd, msg)
	}
}

func AddOutputFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().StringP("output", "o", "yaml", "Output format (json[,multiline]|yaml|text)")
}
