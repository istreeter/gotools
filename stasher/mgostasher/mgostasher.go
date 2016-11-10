package mgostasher

import (
  "context"
  "time"
  "google.golang.org/api/blogger/v3"
  "gopkg.in/mgo.v2"
  "gopkg.in/mgo.v2/bson"
)

const (
  DefaultBlogPageCollection = "blog_page"
  DefaultBlogPostCollection = "blog_post"
  DefaultBlogCollection = "blog"
)

type BlogStasher struct {
  BlogPageCollection *mgo.Collection
  BlogPostCollection *mgo.Collection
  BlogCollection *mgo.Collection
}

type BlogPost struct {
  Id *bson.ObjectId `bson:"_id,omitempty"`
  BlogPost *blogger.Post
  Updated *time.Time
}

type BlogPage struct {
  Id *bson.ObjectId `bson:"_id,omitempty"`
  BlogPage *blogger.Page
  Updated *time.Time
}

type Blog struct {
  Id *bson.ObjectId `bson:"_id,omitempty"`
  Blog *blogger.Blog `bson:",omitempty"`
  Updated *time.Time `bson:",omitempty"`
  PostListEtag *string `bson:",omitempty"`
  PostListUpdated *time.Time `bson:",omitempty"`
  PageListEtag *string `bson:",omitempty"`
  PageListUpdated *time.Time `bson:",omitempty"`
}

func DefaultStasher(db *mgo.Database) (m *BlogStasher) {
  m = new(BlogStasher)
  m.BlogPageCollection = db.C(DefaultBlogPageCollection)
  m.BlogPostCollection = db.C(DefaultBlogPostCollection)
  m.BlogCollection = db.C(DefaultBlogCollection)
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
  _, err := m.BlogPageCollection.Upsert(bson.M{"blogPage.id": page.Id, "updated": bson.M{"$lt": dbPage.Updated}}, &dbPage);
  if err != nil { panic(err) }
}
func(m *BlogStasher) StashPageEtags(ctx context.Context, blogId string, listEtag string, pageEtags map[string]string, newUpdated *time.Time) {
  iter := m.BlogPageCollection.Find(nil).Select(bson.M{"dbPage.id": 1, "dbPage.etag": 1}).Select(bson.M{"_id": 1}).Iter()
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

  _, err := m.BlogCollection.Upsert(bson.M{"blog.id": blogId, "pageListUpdated": bson.M{"$lt": newUpdated}}, bson.M{"$set": &Blog{PageListEtag: &listEtag, PageListUpdated: newUpdated}})
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
  _, err := m.BlogPostCollection.Upsert(bson.M{"blogPost.id": post.Id, "updated": bson.M{"$lt": dbPost.Updated}}, dbPost);
  if err != nil { panic(err) }
}
func(m *BlogStasher) StashPostEtags(ctx context.Context, blogId string, listEtag string, postEtags map[string]string, newUpdated *time.Time) {
  iter := m.BlogPostCollection.Find(nil).Select(bson.M{"dbPost.id": 1, "dbPost.etag": 1}).Select(bson.M{"_id": 1}).Iter()
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

  _, err := m.BlogCollection.Upsert(bson.M{"blog.id": blogId, "postListUpdated": bson.M{"$lt": newUpdated}}, bson.M{"$set": &Blog{PostListEtag: &listEtag, PostListUpdated: newUpdated}})
  if (err != nil) { panic(err) }
}

func(m *BlogStasher) StashBlog(ctx context.Context, blog *blogger.Blog) {
  dbBlog := Blog{
    Blog: blog,
  }
  var err error
  *dbBlog.Updated, err = time.Parse(time.RFC3339, blog.Updated)
  if err != nil { panic(err) }
  _, err = m.BlogCollection.Upsert(bson.M{"blog.id": blog.Id, "updated": bson.M{"$lt": dbBlog.Updated}}, bson.M{"$set": &dbBlog})
  if (err != nil) { panic(err) }
}
