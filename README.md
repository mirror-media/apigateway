# apigateway

## Preliminaries

1. Go 1.16+
2. Run `go mod download` in the directory of `APIGATEWAY`

## Build

Run `make all` or `make clean all` to compile the binaries, `apigateway` and `membermutation` in `bin/`.

## Run

Configurations file names are *implicitly* hardcoded in `cmd` via `vipers`

1. Prepare `config.yaml` for `apigateway` in the `configs` diractory.
2. Prepare `membermutationConfig.yaml` for `membermutation` in the `configs` diractory.
3. Prepare the firebase credential and set the location in the config files. It will be used for verify firebase token.

## Responsibility

Provides APIs for:

1. Verify token
2. Content delivery according to the member subscription state and content state
3. Expose a GraphQL service for the queries and mutations of members and subscriptions

## Design

Architecture-wise, please check `doc/infra.png` to see how it interact with other services.

![infra diagram](https://github.com/mirror-media/apigateway/blob/61a180a336d70eb4cf6b1976d750783ac980efa3/doc/infra.png)

### CMD

APIGATEWAY composites two process, `apigateway` and `membermutation`, which are located in the `cmd` folder. The handling of GraphQL mutation falls to `membermutation`. Anything else is the responsibility of `apigateway`, which is also the entrypoint of the whole service and relay GraphQL mutation to `membermutation`.

### Endpoints

`apigateway` provides the following endpoints

1. `/api/v2/graphql/member` as the GraphQL endpoint
2. `/api/v1/tokenState` as a simple token verification endpoint
3. `/api/v0/*`, any requests coming through it will be proxied to the `restful service` in k8s. 
   1. `/api/v0/story`, requests will be treated as content requests and proxied as a `getposts` request. The response would be truncated if the content is premium
   2. `/api/v0/getposts`, `/api/v0/posts`, and `/api/v0/post`, requests are treated content requests. The response would be truncated if the content is premium and the member has no premium privilege

### Routes and middlewares

1. Route functions can be found in `server/route.go`.
2. There several crucial middlewares
   1. `SetIDTokenOnly` to parse the token and save it to gin.Context
   2. `SetUserID` to parse the userID, i.e., Firebase ID, of the token and save it to gin.Context too
   3. `AuthenticateIDToken` to verify the token status and reject the request if it's not valid
   4. `AuthenticateMemberQueryAndFirebaseIDInArguments` to verify the firebaseID in the where filter of GraphQL requests against the token

### GraphQL Schema

#### UGLY PART

The schema is *manually maintained*. It's defined as in `graph/member/type.graphql`, `graph/member/query.graphql`, and `graph/member/mutation.graphql`.

The usage of pathes is *hardcoded* in routes.

Modification of them is a pain in the something.

#### How/when to use/change them

1. `apigateway` stitches `graph/member/type.graphql`, and `graph/member/query.graphql` as the first schema. Sends related GraphQL requests to `Member GraphQL Service`. They needs to be in sync with `Member GraphQL Service`.
2. `apigateway` stitches `graph/member/type.graphql`, and `graph/member/mutation.graphql` as the second schema. Sends related GraphQL requests to `membermutation`. The implementation mutations are in `graph/member/mutationgraph/mutation.resolvers.go`

When any of the `*.graphql` files are changed, `apigateway` automatically picks them up. However, the implementation of mutations needs to be updated too.

To update them, run `go run github.com/99designs/gqlgen generate` in `graph/member` to generate the boilerplate of go codes and update the signatures in `graph/member/mutationgraph/mutation.resolvers.go`

Implementation can be updated manually after it.

## Known Issues:
- ~~#68: "null" is a keyworkd and can't be used in graphql variables due to a hack in PR#66~~ (Fiexed in #70)
- an empty array of input is parsed as null #95
