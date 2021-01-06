package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"strings"

	"github.com/machinebox/graphql"
	"github.com/mirror-media/apigateway/graph/generated"
	"github.com/mirror-media/apigateway/graph/model"
)

func (r *mutationResolver) Member(ctx context.Context) (*model.MemberType, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) CreateMember(ctx context.Context, email string, firebaseID string, nickname *string) (*model.CreateMember, error) {
	if _, err := r.IsRequestMatchingRequesterFirebaseID(ctx, firebaseID); err != nil {
		return nil, err
	}

	// Construct GraphQL mutation
	preloads := GetPreloads(ctx)
	preGQL := []string{"mutation($email: String!, $firebaseId: String!, $nickname: String) {", "createMember(email: $email, firebaseId: $firebaseId, nickname: $nickname) {"}

	fieldsOnly := Map(preloads, func(s string) string {
		ns := strings.Split(s, ".")
		return ns[len(ns)-1]
	})

	preGQL = append(preGQL, fieldsOnly...)
	preGQL = append(preGQL, "}", "}")
	gql := strings.Join(preGQL, "\n")

	req := graphql.NewRequest(gql)
	req.Var("firebaseId", firebaseID)
	req.Var("email", email)
	req.Var("nickname", nickname)

	// Ask User service to create the member
	var resp struct {
		CreateMember *model.CreateMember `json:"createMember"`
	}
	err := graphql.NewClient(r.Resolver.UserSrvURL).Run(ctx, req, &resp)

	return resp.CreateMember, err
}

func (r *mutationResolver) UpdateMember(ctx context.Context, address *string, birthday *string, city *string, country *string, district *string, firebaseID string, gender *int, name *string, nickname *string, phone *string, profileImage *string) (*model.UpdateMember, error) {
	if _, err := r.IsRequestMatchingRequesterFirebaseID(ctx, firebaseID); err != nil {
		return nil, err
	}

	// Construct GraphQL mutation
	preloads := GetPreloads(ctx)
	preGQL := []string{"mutation($address: String, $birthday: Date, $city: String, $country: String, $district: String, $firebaseId: String!, $gender: Int, $name: String, $nickname: String, $phone: String, $profileImage: String) {", "updateMember(address: $address, birthday: $birthday, city: $city, country: $country, district: $district, firebaseId: $firebaseId, gender: $gender, name: $name, nickname: $nickname, phone: $phone, profileImage: $profileImage) {"}

	fieldsOnly := Map(preloads, func(s string) string {
		ns := strings.Split(s, ".")
		return ns[len(ns)-1]
	})

	preGQL = append(preGQL, fieldsOnly...)
	preGQL = append(preGQL, "}", "}")
	gql := strings.Join(preGQL, "\n")

	req := graphql.NewRequest(gql)
	req.Var("firebaseId", firebaseID)
	req.Var("address", address)
	req.Var("birthday", birthday)
	req.Var("city", city)
	req.Var("country", country)
	req.Var("district", district)
	req.Var("firebaseId", firebaseID)
	req.Var("gender", gender)
	req.Var("name", name)
	req.Var("nickname", nickname)
	req.Var("phone", phone)
	req.Var("profileImage", profileImage)

	// Ask User service to update the member
	var resp struct {
		UpdateMember *model.UpdateMember `json:"updateMember"`
	}
	err := graphql.NewClient(r.Resolver.UserSrvURL).Run(ctx, req, &resp)

	return resp.UpdateMember, err
}

func (r *mutationResolver) DeleteMember(ctx context.Context, firebaseID string) (*model.DeleteMember, error) {
	if _, err := r.IsRequestMatchingRequesterFirebaseID(ctx, firebaseID); err != nil {
		return nil, err
	}
	// delete Firebase user
	client, err := FirebaseClientFromContext(ctx)
	if err != nil {
		return nil, err
	}

	err = client.DeleteUser(ctx, firebaseID)

	// WIP remove the request and use pub/sub
	if err != nil {
		return nil, err
	}

	// Construct GraphQL mutation
	preloads := GetPreloads(ctx)
	preGQL := []string{"mutation($firebaseId: String!) {", "deleteMember(firebaseId: $firebaseId) {"}

	fieldsOnly := Map(preloads, func(s string) string {
		ns := strings.Split(s, ".")
		return ns[len(ns)-1]
	})

	preGQL = append(preGQL, fieldsOnly...)
	preGQL = append(preGQL, "}", "}")
	gql := strings.Join(preGQL, "\n")

	req := graphql.NewRequest(gql)
	req.Var("firebaseId", firebaseID)

	// Ask User service to delete the member
	var resp struct {
		DeleteMember *model.DeleteMember `json:"deleteMember"`
	}
	err = graphql.NewClient(r.Resolver.UserSrvURL).Run(ctx, req, &resp)

	return resp.DeleteMember, err
}

func (r *mutationResolver) VerifyMember(ctx context.Context, token string) (*model.VerifyAccount, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) ArchiveAccount(ctx context.Context, password string) (*model.ArchiveAccount, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) SendSecondaryEmailActivation(ctx context.Context, email string, password string) (*model.SendSecondaryEmailActivation, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) VerifySecondaryEmail(ctx context.Context, token string) (*model.VerifySecondaryEmail, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) SwapEmails(ctx context.Context, password string) (*model.SwapEmails, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) TokenAuth(ctx context.Context, password string, email *string, username *string) (*model.ObtainJSONWebToken, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) VerifyToken(ctx context.Context, token string) (*model.VerifyToken, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) RefreshToken(ctx context.Context, refreshToken string) (*model.RefreshToken, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) RevokeToken(ctx context.Context, refreshToken string) (*model.RevokeToken, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) Member(ctx context.Context, firebaseID string) (*model.MemberType, error) {
	if _, err := r.IsRequestMatchingRequesterFirebaseID(ctx, firebaseID); err != nil {
		return nil, err
	}

	preloads := GetPreloads(ctx)

	client := graphql.NewClient(r.Resolver.UserSrvURL)

	// Construct GraphQL Query
	preGQL := []string{"query ($firebaseId: String!) {", "member(firebaseId: $firebaseId) {"}

	fieldsOnly := Map(preloads, func(s string) string {
		ns := strings.Split(s, ".")
		return ns[len(ns)-1]
	})

	preGQL = append(preGQL, fieldsOnly...)
	preGQL = append(preGQL, "}", "}")
	gql := strings.Join(preGQL, "\n")

	// Do the query
	req := graphql.NewRequest(gql)
	req.Var("firebaseId", firebaseID)
	var member struct {
		Member *model.MemberType `json:"member"`
	}
	err := client.Run(ctx, req, &member)

	return member.Member, err
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
