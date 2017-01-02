package mgostasher

import (
  "context"
  "time"
  "google.golang.org/api/blogger/v3"
  "github.com/dghubble/go-twitter/twitter"
  "gopkg.in/mgo.v2"
  "gopkg.in/mgo.v2/bson"
)

const (
  DefaultBlogPageCollection = "blog_page"
  DefaultBlogPostCollection = "blog_post"
  DefaultBlogCollection = "blog"
  DefaultTweetCollection = "tweet"
  DefaultTwitterUserCollection = "twitterer"
)

type BlogStasher struct {
  BlogPageCollection *mgo.Collection
  BlogPostCollection *mgo.Collection
  BlogCollection *mgo.Collection
}

type TwitterStasher struct {
  TweetCollection *mgo.Collection
  TwitterUserCollection *mgo.Collection
}

type BlogPost struct {
  Id *bson.ObjectId `bson:"_id,omitempty" json:"id,omitempty"`
  BlogPost *blogger.Post `bson:"blogPost" json:"blogPost"`
  Updated *time.Time `bson:"updated" json:"updated"`
}

type BlogPage struct {
  Id *bson.ObjectId `bson:"_id,omitempty" json:"id,omitempty"`
  BlogPage *blogger.Page `bson:"blogPage" json:"blogPage"`
  Updated *time.Time `bson:"updated" json:"updated"`
}

type Blog struct {
  Id *bson.ObjectId `bson:"_id,omitempty" json:"id,omitempty"`
  Blog *blogger.Blog `bson:"blog,omitempty" json:"blog"`
  Updated *time.Time `bson:"updated,omitempty" json:"updated"`
  PostListEtag *string `bson:"postListEtag,omitempty"`
  PostListUpdated *time.Time `bson:"postListUpdated,omitempty"`
  PageListEtag *string `bson:"pageListEtag,omitempty"`
  PageListUpdated *time.Time `bson:"pageListUpdated,omitempty"`
}

type Tweet struct {
  Id *bson.ObjectId `bson:"_id,omitempty" json:",omitempty"`
  Tweet *twitter.Tweet `bson:"tweet"`
  OEmbed *twitter.OEmbedTweet `bson:"oEmbed"`
}

type TwitterUser struct {
  Id *bson.ObjectId `bson:"_id,omitempty" json:",omitempty"`
  TwitterUser *twitter.User `bson:"twitterUser"`
}

func DefaultBlogStasher(db *mgo.Database) (m *BlogStasher) {
  m = new(BlogStasher)
  m.BlogPageCollection = db.C(DefaultBlogPageCollection)
  m.BlogPostCollection = db.C(DefaultBlogPostCollection)
  m.BlogCollection = db.C(DefaultBlogCollection)
  return
}

func DefaultTwitterStasher(db *mgo.Database) (m *TwitterStasher) {
  m = new(TwitterStasher)
  m.TweetCollection = db.C(DefaultTweetCollection)
  m.TwitterUserCollection = db.C(DefaultTwitterUserCollection)
  return
}

func(m *BlogStasher) GetPageListEtag(ctx context.Context, blogId string) (etag *string) {
  dbBlog := new(Blog)
  err := m.BlogCollection.Find(bson.M{"blog.id": blogId}).Select(bson.M{"pageListEtag": 1}).One(dbBlog)
  if err != nil {
    if err == mgo.ErrNotFound {
      return nil
    }
    panic(err)
  }
  return dbBlog.PageListEtag
}
func(m *BlogStasher) HasPage(ctx context.Context, pageId string, etag string) bool {
  n, err := m.BlogPageCollection.Find(bson.M{"blogPage.id": pageId, "blogPage.etag": etag}).Count()
  if err != nil { panic(err) }
  if n > 0 { return true }
  return false
}
func WrapBlogPage(page *blogger.Page) *BlogPage {
  dbPage := &BlogPage{
    BlogPage: page,
  }
  pageUpdated, err := time.Parse(time.RFC3339, page.Updated)
  if err != nil { panic(err) }
  dbPage.Updated = &pageUpdated
  return dbPage
}
func(m *BlogStasher) StashPage(ctx context.Context, page *blogger.Page) {
  dbPage := WrapBlogPage(page)
  _, err := m.BlogPageCollection.Upsert(bson.M{"blogPage.id": page.Id}, &dbPage);
  if err != nil { panic(err) }
}
func(m *BlogStasher) StashPageEtags(ctx context.Context, blogId string, listEtag string, pageEtags map[string]string, newUpdated *time.Time) {
  iter := m.BlogPageCollection.
      Find(nil).
      Select(bson.M{"blogPage.id": 1, "blogPage.etag": 1}).
      Iter()
  defer iter.Close()
  dbPage := new(BlogPage)
  for iter.Next(dbPage) {
    if pageEtags[dbPage.BlogPage.Id] != dbPage.BlogPage.Etag {
      if err := m.BlogPageCollection.RemoveId(dbPage.Id); err != nil {
        panic(err)
      }
    }
  }
  if err := iter.Err(); err != nil {
    panic(err)
  }

  _, err := m.BlogCollection.Upsert(bson.M{"blog.id": blogId}, bson.M{"$set": &Blog{PageListEtag: &listEtag, PageListUpdated: newUpdated}})
  if (err != nil) { panic(err) }
}

func(m *BlogStasher) GetPostListEtag(ctx context.Context, blogId string) (etag *string) {
  dbBlog := new(Blog)
  err := m.BlogCollection.Find(bson.M{"blog.id": blogId}).Select(bson.M{"postListEtag": 1}).One(dbBlog)
  if err != nil {
    if err == mgo.ErrNotFound {
      return nil
    }
    panic(err)
  }
  return dbBlog.PostListEtag
}
func(m *BlogStasher) HasPost(ctx context.Context, postId string, etag string) bool {
  n, err := m.BlogPostCollection.Find(bson.M{"blogPost.id": postId, "blogPost.etag": etag}).Count()
  if err != nil { panic(err) }
  if n > 0 { return true }
  return false
}

func WrapBlogPost(post *blogger.Post) *BlogPost {
  dbPost := &BlogPost{
    BlogPost: post,
  }
  postUpdated, err := time.Parse(time.RFC3339, post.Updated)
  if err != nil { panic(err) }
  dbPost.Updated = &postUpdated
  return dbPost
}
func(m *BlogStasher) StashPost(ctx context.Context, post *blogger.Post) {
  dbPost := WrapBlogPost(post)
  _, err := m.BlogPostCollection.Upsert(bson.M{"blogPost.id": post.Id}, dbPost);
  if err != nil { panic(err) }
}
func(m *BlogStasher) StashPostEtags(ctx context.Context, blogId string, listEtag string, postEtags map[string]string, newUpdated *time.Time) {
  iter := m.BlogPostCollection.
      Find(nil).
      Select(bson.M{"blogPost.id": 1, "blogPost.etag": 1}).
      Iter()
  defer iter.Close()
  dbPost := new(BlogPost)
  for iter.Next(dbPost) {
    if postEtags[dbPost.BlogPost.Id] != dbPost.BlogPost.Etag {
      if err := m.BlogPostCollection.RemoveId(dbPost.Id); err != nil {
        panic(err)
      }
    }
  }
  if err := iter.Err(); err != nil {
    panic(err)
  }

  _, err := m.BlogCollection.Upsert(bson.M{"blog.id": blogId}, bson.M{"$set": &Blog{PostListEtag: &listEtag, PostListUpdated: newUpdated}})
  if (err != nil) { panic(err) }
}

func(m *BlogStasher) StashBlog(ctx context.Context, blog *blogger.Blog) {
  dbBlog := Blog{
    Blog: blog,
  }
  updated, err := time.Parse(time.RFC3339, blog.Updated)
  if err != nil { panic(err) }
  dbBlog.Updated = &updated
  _, err = m.BlogCollection.Upsert(bson.M{"blog.id": blog.Id}, bson.M{"$set": &dbBlog})
  if (err != nil) { panic(err) }
}

func(m *TwitterStasher) GetLastTweetId(ctx context.Context, userId int64) int64 {
  tweet := new(Tweet)
  err := m.TweetCollection.Find(bson.M{"tweet.user.id": userId}).Select(bson.M{"tweet.id": 1}).One(tweet)
  if err != nil {
    if err == mgo.ErrNotFound {
      return 0
    }
    panic(err)
  }
  return tweet.Tweet.ID
}

func(m *TwitterStasher) StashUser(ctx context.Context, user *twitter.User) {
  dbUser := &TwitterUser{TwitterUser: user}
  _, err := m.TwitterUserCollection.Upsert(bson.M{"twitterUser.id": user.ID}, &dbUser);
  if err != nil { panic(err) }
}

func(m *TwitterStasher) StashTweet(ctx context.Context, tweet *twitter.Tweet, oembed *twitter.OEmbedTweet) {
  dbTweet := &Tweet{Tweet: tweet, OEmbed: oembed}
  _, err := m.TweetCollection.Upsert(bson.M{"tweet.id": tweet.ID}, &dbTweet);
  if err != nil { panic(err) }
}
