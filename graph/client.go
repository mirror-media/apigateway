package graph

import (
	"context"

	"github.com/machinebox/graphql"
)

type Client struct {
	*graphql.Client
}

func (c *Client) Run(ctx context.Context, req *graphql.Request, resp interface{}) (err error) {

	err = c.Client.Run(ctx, req, resp)
	switch err.Error() {
	case "graphql: Invalid token":
		// refresh token
	}

	return err
}
