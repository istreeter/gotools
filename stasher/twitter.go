package stasher

import (
  "context"
  "github.com/dghubble/go-twitter/twitter"
  "golang.org/x/oauth2"
  "fmt"
  "net/http"
  "log"
  "time"
)

type TwitterStasher interface {
  GetLastTweetId(ctx context.Context, userId int64) int64
  StashUser(context.Context, *twitter.User)
  StashTweet(context.Context, *twitter.Tweet, *twitter.OEmbedTweet)
}

type twitterGetter interface {
  user(context.Context, int64) *twitter.User
  userTimeline(context.Context, *twitter.UserTimelineParams) []twitter.Tweet
  oembed(context.Context, *twitter.Tweet) *twitter.OEmbedTweet
}

func SyncTwitter(ctx context.Context, userId int64, stasher TwitterStasher, accessToken string) (err error) {
  defer func() {
    if r := recover(); r != nil {
      var ok bool
      err, ok = r.(error)
      if !ok {
        err = fmt.Errorf("pkg: %v", r)
      }
    }
  }()

  config := &oauth2.Config{}
  token := &oauth2.Token{AccessToken: accessToken}
  httpClient := config.Client(ctx, token)
  twitterClient := (*twitterClient)(twitter.NewClient(httpClient))

  syncTwitter(ctx, userId, stasher, twitterClient)
  return nil
}

type twitterClient twitter.Client;

func (client *twitterClient) user(ctx context.Context, userId int64) *twitter.User {
  c := (*twitter.Client)(client)
  var user *twitter.User
  var err error
  for try := 0; try < 2; try++ {
    var res *http.Response
    user, res, err = c.Users.Show(&twitter.UserShowParams{UserID: userId})
    if res != nil && res.StatusCode == 420 || res.StatusCode == 429 {
      log.Println("Rate limited by Twitter. Sleeping for 15 minutes...")
      time.Sleep(15 * time.Minute)
      after := time.After(15 * time.Minute)
      select {
        case <-after:
          continue
        case <-ctx.Done():
          panic(ctx.Err())
      }
    }
    break
  }
  if err != nil {
    panic(err)
  }
  return user
}

func (client *twitterClient) userTimeline(ctx context.Context, p *twitter.UserTimelineParams) []twitter.Tweet {
  c := (*twitter.Client)(client)
  var t []twitter.Tweet
  var err error
  for try := 0; try < 2; try++ {
    var res *http.Response
    t, res, err = c.Timelines.UserTimeline(p)
    if res != nil && res.StatusCode == 420 || res.StatusCode == 429 {
      log.Println("Rate limited by Twitter. Sleeping for 15 minutes...")
      after := time.After(15 * time.Minute)
      select {
        case <-after:
          continue
        case <-ctx.Done():
          panic(ctx.Err())
      }
    }
    break
  }
  if err != nil {
    panic(err)
  }
  return t
}

func (client *twitterClient) oembed(ctx context.Context, t *twitter.Tweet) *twitter.OEmbedTweet {
  c := (*twitter.Client)(client)
  var o *twitter.OEmbedTweet
  var err error
  p := &twitter.StatusOEmbedParams{
    ID: t.ID,
    OmitScript: twitter.Bool(true),
    HideThread: twitter.Bool(true),
    HideMedia: twitter.Bool(false),
  }
  for try := 0; try < 2; try++ {
    var res *http.Response
    o, res, err = c.Statuses.OEmbed(p)
    if res != nil && res.StatusCode == 420 || res.StatusCode == 429 {
      log.Println("Rate limited by Twitter. Sleeping for 15 minutes...")
      after := time.After(15 * time.Minute)
      select {
        case <-after:
          continue
        case <-ctx.Done():
          panic(ctx.Err())
      }
    }
    break
  }
  if err != nil {
    panic(err)
  }
  return o
}

func syncTwitter(ctx context.Context, userId int64, stasher TwitterStasher, getter twitterGetter) {
  newUser := getter.user(ctx, userId)
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
    tweets := getter.userTimeline(ctx, params)
    if len(tweets) == 0 {
      break
    }
    for _, tweet := range tweets {
      if tweet.RetweetedStatus != nil {
        continue
      }
      oembed := getter.oembed(ctx, &tweet)
      stasher.StashTweet(ctx, &tweet, oembed)
    }
    lastGotId := tweets[len(tweets)-1].ID
    if params.MaxID > 0 && lastGotId >= params.MaxID {
      continue
    }
    params.MaxID = lastGotId
  }
}
