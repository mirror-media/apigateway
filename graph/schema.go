package graph

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/jensneuse/graphql-go-tools/pkg/graphql"
)

func AlchemizeSchema(filePathes ...string) (*graphql.Schema, error) {

	if len(filePathes) == 0 {
		return nil, fmt.Errorf("file is not provided for the schema")
	}

	var buffer bytes.Buffer
	bufferWriter := bufio.NewWriter(&buffer)

	for _, path := range filePathes {
		fileReader, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer fileReader.Close()

		_, err = io.Copy(bufferWriter, fileReader)
		if err != nil {
			return nil, err
		}
	}

	return graphql.NewSchemaFromString(buffer.String())
}
