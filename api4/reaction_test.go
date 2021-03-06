// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package api4

import (
	"strings"
	"testing"

	"reflect"

	"github.com/mattermost/platform/app"
	"github.com/mattermost/platform/model"
)

func TestSaveReaction(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer TearDown()
	Client := th.Client
	userId := th.BasicUser.Id
	postId := th.BasicPost.Id

	reaction := &model.Reaction{
		UserId:    userId,
		PostId:    postId,
		EmojiName: "smile",
	}

	rr, resp := Client.SaveReaction(reaction)
	CheckNoError(t, resp)

	if rr.UserId != reaction.UserId {
		t.Fatal("UserId did not match")
	}

	if rr.PostId != reaction.PostId {
		t.Fatal("PostId did not match")
	}

	if rr.EmojiName != reaction.EmojiName {
		t.Fatal("EmojiName did not match")
	}

	if rr.CreateAt == 0 {
		t.Fatal("CreateAt should exist")
	}

	if reactions, err := app.GetReactionsForPost(postId); err != nil && len(reactions) != 1 {
		t.Fatal("didn't save reaction correctly")
	}

	// saving a duplicate reaction
	rr, resp = Client.SaveReaction(reaction)
	CheckNoError(t, resp)

	if reactions, err := app.GetReactionsForPost(postId); err != nil && len(reactions) != 1 {
		t.Fatal("should have not save duplicated reaction")
	}

	reaction.EmojiName = "sad"

	rr, resp = Client.SaveReaction(reaction)
	CheckNoError(t, resp)

	if rr.EmojiName != reaction.EmojiName {
		t.Fatal("EmojiName did not match")
	}

	if reactions, err := app.GetReactionsForPost(postId); err != nil && len(reactions) != 2 {
		t.Fatal("should have save multiple reactions")
	}

	reaction.PostId = GenerateTestId()

	_, resp = Client.SaveReaction(reaction)
	CheckForbiddenStatus(t, resp)

	reaction.PostId = "junk"

	_, resp = Client.SaveReaction(reaction)
	CheckBadRequestStatus(t, resp)

	reaction.PostId = postId
	reaction.UserId = GenerateTestId()

	_, resp = Client.SaveReaction(reaction)
	CheckForbiddenStatus(t, resp)

	reaction.UserId = "junk"

	_, resp = Client.SaveReaction(reaction)
	CheckBadRequestStatus(t, resp)

	reaction.UserId = userId
	reaction.EmojiName = ""

	_, resp = Client.SaveReaction(reaction)
	CheckBadRequestStatus(t, resp)

	reaction.EmojiName = strings.Repeat("a", 65)

	_, resp = Client.SaveReaction(reaction)
	CheckBadRequestStatus(t, resp)

	reaction.EmojiName = "smile"
	otherUser := th.CreateUser()
	Client.Logout()
	Client.Login(otherUser.Email, otherUser.Password)

	_, resp = Client.SaveReaction(reaction)
	CheckForbiddenStatus(t, resp)

	Client.Logout()
	_, resp = Client.SaveReaction(reaction)
	CheckUnauthorizedStatus(t, resp)

	_, resp = th.SystemAdminClient.SaveReaction(reaction)
	CheckForbiddenStatus(t, resp)
}

func TestGetReactions(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer TearDown()
	Client := th.Client
	userId := th.BasicUser.Id
	user2Id := th.BasicUser2.Id
	postId := th.BasicPost.Id

	userReactions := []*model.Reaction{
		{
			UserId:    userId,
			PostId:    postId,
			EmojiName: "smile",
		},
		{
			UserId:    userId,
			PostId:    postId,
			EmojiName: "happy",
		},
		{
			UserId:    userId,
			PostId:    postId,
			EmojiName: "sad",
		},
		{
			UserId:    user2Id,
			PostId:    postId,
			EmojiName: "smile",
		},
		{
			UserId:    user2Id,
			PostId:    postId,
			EmojiName: "sad",
		},
	}

	var reactions []*model.Reaction

	for _, userReaction := range userReactions {
		if result := <-app.Srv.Store.Reaction().Save(userReaction); result.Err != nil {
			t.Fatal(result.Err)
		} else {
			reactions = append(reactions, result.Data.(*model.Reaction))
		}
	}

	rr, resp := Client.GetReactions(postId)
	CheckNoError(t, resp)

	if len(rr) != 5 {
		t.Fatal("reactions should returned correct length")
	}

	if !reflect.DeepEqual(rr, reactions) {
		t.Fatal("reactions should have matched")
	}

	rr, resp = Client.GetReactions("junk")
	CheckBadRequestStatus(t, resp)

	if len(rr) != 0 {
		t.Fatal("reactions should return empty")
	}

	_, resp = Client.GetReactions(GenerateTestId())
	CheckForbiddenStatus(t, resp)

	Client.Logout()

	_, resp = Client.GetReactions(postId)
	CheckUnauthorizedStatus(t, resp)

	_, resp = th.SystemAdminClient.GetReactions(postId)
	CheckNoError(t, resp)
}

func TestDeleteReaction(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer TearDown()
	Client := th.Client
	userId := th.BasicUser.Id
	user2Id := th.BasicUser2.Id
	postId := th.BasicPost.Id

	r1 := &model.Reaction{
		UserId:    userId,
		PostId:    postId,
		EmojiName: "smile",
	}

	app.SaveReactionForPost(r1)
	if reactions, err := app.GetReactionsForPost(postId); err != nil || len(reactions) != 1 {
		t.Fatal("didn't save reaction correctly")
	}

	ok, resp := Client.DeleteReaction(r1)
	CheckNoError(t, resp)

	if !ok {
		t.Fatal("should have returned true")
	}

	if reactions, err := app.GetReactionsForPost(postId); err != nil || len(reactions) != 0 {
		t.Fatal("should have deleted reaction")
	}

	// deleting one reaction when a post has multiple reactions
	r2 := &model.Reaction{
		UserId:    userId,
		PostId:    postId,
		EmojiName: "smile-",
	}

	app.SaveReactionForPost(r1)
	app.SaveReactionForPost(r2)
	if reactions, err := app.GetReactionsForPost(postId); err != nil || len(reactions) != 2 {
		t.Fatal("didn't save reactions correctly")
	}

	_, resp = Client.DeleteReaction(r2)
	CheckNoError(t, resp)

	if reactions, err := app.GetReactionsForPost(postId); err != nil || len(reactions) != 1 || *reactions[0] != *r1 {
		t.Fatal("should have deleted 1 reaction only")
	}

	// deleting a reaction made by another user
	r3 := &model.Reaction{
		UserId:    user2Id,
		PostId:    postId,
		EmojiName: "smile_",
	}

	th.LoginBasic2()
	app.SaveReactionForPost(r3)
	if reactions, err := app.GetReactionsForPost(postId); err != nil || len(reactions) != 2 {
		t.Fatal("didn't save reaction correctly")
	}

	th.LoginBasic()

	ok, resp = Client.DeleteReaction(r3)
	CheckForbiddenStatus(t, resp)

	if ok {
		t.Fatal("should have returned false")
	}

	if reactions, err := app.GetReactionsForPost(postId); err != nil || len(reactions) != 2 {
		t.Fatal("should have not deleted a reaction")
	}

	r1.PostId = GenerateTestId()
	_, resp = Client.DeleteReaction(r1)
	CheckForbiddenStatus(t, resp)

	r1.PostId = "junk"

	_, resp = Client.DeleteReaction(r1)
	CheckBadRequestStatus(t, resp)

	r1.PostId = postId
	r1.UserId = GenerateTestId()

	_, resp = Client.DeleteReaction(r1)
	CheckForbiddenStatus(t, resp)

	r1.UserId = "junk"

	_, resp = Client.DeleteReaction(r1)
	CheckBadRequestStatus(t, resp)

	r1.UserId = userId
	r1.EmojiName = ""

	_, resp = Client.DeleteReaction(r1)
	CheckNotFoundStatus(t, resp)

	r1.EmojiName = strings.Repeat("a", 65)

	_, resp = Client.DeleteReaction(r1)
	CheckBadRequestStatus(t, resp)

	Client.Logout()
	r1.EmojiName = "smile"

	_, resp = Client.DeleteReaction(r1)
	CheckUnauthorizedStatus(t, resp)

	_, resp = th.SystemAdminClient.DeleteReaction(r1)
	CheckNoError(t, resp)

	_, resp = th.SystemAdminClient.DeleteReaction(r3)
	CheckNoError(t, resp)

	if reactions, err := app.GetReactionsForPost(postId); err != nil || len(reactions) != 0 {
		t.Fatal("should have deleted both reactions")
	}
}
