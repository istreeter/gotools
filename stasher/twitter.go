package stasher

import (
  "context"
  "github.com/dghubble/go-twitter/twitter"
  //"golang.org/x/oauth2"
)

type TwitterStasher interface {
  GetLastTweetId(ctx context.Context, userId int64) int64
  StashUser(context.Context, *twitter.User)
  StashTweet(context.Context, *twitter.Tweet, *twitter.OEmbedTweet)
}

type twitterGetter interface {
  user(int64) *twitter.User
  userTimeline(*twitter.UserTimelineParams) []twitter.Tweet
  oembed(*twitter.Tweet) *twitter.OEmbedTweet
}

func syncTwitter(ctx context.Context, userId int64, stasher TwitterStasher, getter twitterGetter) {
  newUser := getter.user(userId)
  stasher.StashUser(ctx, newUser)
  lastTweetId := stasher.GetLastTweetId(ctx, userId)
  params := &twitter.UserTimelineParams{
    SinceID: lastTweetId,
    UserID: userId,
    TrimUser: twitter.Bool(true),
    IncludeRetweets: twitter.Bool(true),
    ExcludeReplies: twitter.Bool(true),
    Count: 50,
  }
  for {
    tweets := getter.userTimeline(params)
    if len(tweets) == 0 {
      break
    }
    for _, tweet := range tweets {
      if tweet.RetweetedStatus != nil {
        continue
      }
      oembed := getter.oembed(&tweet)
      stasher.StashTweet(ctx, &tweet, oembed)
    }
    lastGotId := tweets[len(tweets)-1].ID
    if params.MaxID > 0 && lastGotId >= params.MaxID {
      continue
    }
    params.MaxID = lastGotId
  }
}
