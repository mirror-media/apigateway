package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"strings"

	"github.com/machinebox/graphql"
	"github.com/mirror-media/mm-apigateway/graph/generated"
	"github.com/mirror-media/mm-apigateway/graph/model"
	"github.com/mirror-media/mm-apigateway/member"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var logger = logrus.New().WithField("type", "graphQL")

func (r *mutationResolver) TokenCreate(ctx context.Context, password string, email *string, username *string) (*model.ObtainJSONWebToken, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) TokenRefresh(ctx context.Context, refreshToken string) (*model.RefreshToken, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) TokenVerify(ctx context.Context, token string) (*model.VerifyToken, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) Member(ctx context.Context) (*model.Member, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) CreateMember(ctx context.Context, email *string, firebaseID string) (*model.CreateMember, error) {
	if _, err := r.IsRequestMatchingRequesterFirebaseID(ctx, firebaseID); err != nil {
		return nil, err
	}

	// Construct GraphQL mutation
	preloads := GetPreloads(ctx)
	preGQL := []string{"mutation($email: String, $firebaseId: String!) {", "createMember(email: $email, firebaseId: $firebaseId) {"}

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

	// Ask User service to create the member
	var resp struct {
		CreateMember *model.CreateMember `json:"createMember"`
	}
	err := r.Client.Run(ctx, req, &resp)

	checkAndPrintGraphQLError(logger.WithField("mutation", "CreateMember"), err)

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
	err := r.Client.Run(ctx, req, &resp)

	checkAndPrintGraphQLError(logger.WithField("mutation", "UpdateMember"), err)

	return resp.UpdateMember, err
}

func (r *mutationResolver) DeleteMember(ctx context.Context, firebaseID string) (*model.DeleteMember, error) {
	if _, err := r.IsRequestMatchingRequesterFirebaseID(ctx, firebaseID); err != nil {
		return nil, err
	}
	client, err := FirebaseClientFromContext(ctx)
	if err != nil {
		errors.WithMessage(err, "can't get FirebaseClient from context")
		logrus.Error(err)
		return nil, err
	}

	// disable firebase user before delete member to decrease response time
	if err = member.DisableFirebaseUser(ctx, client, firebaseID); err != nil {
		errors.WithMessage(err, fmt.Sprintf("can't disable Firebaseuser(%s)", firebaseID))
		logrus.Error(err)
		return nil, err
	}

	dbClient, err := FirebaseDatabaseClientFromContext(ctx)
	if err != nil {
		errors.WithMessage(err, "can't get FirebaseDatabaseClient from context")
		logrus.Error(err)
		return nil, err
	}

	// delete Firebase user and request to disable member in DB concurrently
	// use context.Background() so that the "delete member" can finish without interuption
	go func() {
		err = member.Delete(context.Background(), r.Conf, client, dbClient, firebaseID)
		if err != nil {
			err = errors.WithMessagef(err, "Failed to delete Firebase User(%s) or publish to delete the member", firebaseID)
			logrus.Error(err)
		}
	}()

	Success := true
	logrus.Infof("Successfully disable the Firebase user(%s)", firebaseID)
	return &model.DeleteMember{
		Success: &Success,
	}, err
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

func (r *queryResolver) Member(ctx context.Context, firebaseID string) (*model.Member, error) {
	if _, err := r.IsRequestMatchingRequesterFirebaseID(ctx, firebaseID); err != nil {
		return nil, err
	}

	preloads := GetPreloads(ctx)

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
		Member *model.Member `json:"member"`
	}
	err := r.Client.Run(ctx, req, &member)

	checkAndPrintGraphQLError(logger.WithField("query", "Member"), err)

	return member.Member, err
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
