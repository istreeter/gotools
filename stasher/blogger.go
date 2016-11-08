package stasher

import (
  "context"
  "time"
  "net/http"
  "fmt"
  "google.golang.org/api/blogger/v3"
  "google.golang.org/api/googleapi"
  "golang.org/x/oauth2/google"
  "gopkg.in/mgo.v2"
  "gopkg.in/mgo.v2/bson"
)

const (
  DefaultBlogPageCollection = "blog_page"
  DefaultBlogPostCollection = "blog_post"
  DefaultBlogCollection = "blog"
)

type BloggerStasher interface {
  BlogPostStasher
  BlogPageStasher
  BlogStasher
}

type BlogPostStasher interface {
  GetPostListEtag(ctx context.Context, blogId string) (etag *string)
  HasPost(ctx context.Context, pageId string, etag string) bool
  StashPost(context.Context, *blogger.Post)
  StashPostEtags(ctx context.Context, blogId string, listEtag string, postEtags map[string]string, updated *time.Time)
}

type BlogPageStasher interface {
  GetPageListEtag(ctx context.Context, blogId string) (etag *string)
  HasPage(ctx context.Context, pageId string, etag string) bool
  StashPage(context.Context, *blogger.Page)
  StashPageEtags(ctx context.Context, blogId string, listEtag string, pageEtags map[string]string, updated *time.Time)
}

type BlogStasher interface {
  StashBlog(context.Context, *blogger.Blog)
}

type MgoBlogStasher struct {
  BlogPageCollection *mgo.Collection
  BlogPostCollection *mgo.Collection
  BlogCollection *mgo.Collection
}

type MgoBlogPost struct {
  Id *bson.ObjectId `bson:"_id,omitempty"`
  BlogPost *blogger.Post
  Updated *time.Time
}

type MgoBlogPage struct {
  Id *bson.ObjectId `bson:"_id,omitempty"`
  BlogPage *blogger.Page
  Updated *time.Time
}

type MgoBlog struct {
  Id *bson.ObjectId `bson:"_id,omitempty"`
  Blog *blogger.Blog `bson:",omitempty"`
  Updated *time.Time `bson:",omitempty"`
  PostListEtag *string `bson:",omitempty"`
  PostListUpdated *time.Time `bson:",omitempty"`
  PageListEtag *string `bson:",omitempty"`
  PageListUpdated *time.Time `bson:",omitempty"`
}

func DefaultMgoStasher(db *mgo.Database) (m *MgoBlogStasher) {
  m = new(MgoBlogStasher)
  m.BlogPageCollection = db.C(DefaultBlogPageCollection)
  m.BlogPostCollection = db.C(DefaultBlogPostCollection)
  m.BlogCollection = db.C(DefaultBlogCollection)
  return
}

type blogPostGetter interface {
  blogPostEtags(blogId string, oldListEtag *string) (newListEtag string, newPostEtags map[string]string)
  blogPost(blogId string, postId string) *blogger.Post
}

type blogPageGetter interface {
  blogPageEtags(blogId string, oldListEtag *string) (newListEtag string, newPageEtags map[string]string)
  blogPage(blogId string, pageId string) *blogger.Page
}
type blogGetter interface {
  blog(blogId string) *blogger.Blog
}

type blogService blogger.Service;

func(b *blogService) blogPostEtags(blogId string, oldListEtag *string) (newListEtag string, newPostEtags map[string]string) {
  s := (*blogger.Service)(b)
  postListCall := s.Posts.List(blogId).FetchBodies(false)
  if oldListEtag != nil {
    postListCall = postListCall.IfNoneMatch(*oldListEtag)
  }
  newPostEtags = make(map[string]string)
  err := postListCall.Pages(nil, func(pList *blogger.PostList) error {
    newListEtag = pList.Etag
    for _, p := range pList.Items {
      newPostEtags[p.Id] = p.Etag
    }
    return nil
  })
  if err != nil && ! googleapi.IsNotModified(err) {
    panic(err)
  }
  return
}

func(b *blogService) blogPageEtags(blogId string, oldListEtag *string) (newListEtag string, newPageEtags map[string]string) {
  s := (*blogger.Service)(b)
  pageListCall := s.Pages.List(blogId).FetchBodies(false)
  if oldListEtag != nil {
    pageListCall = pageListCall.IfNoneMatch(*oldListEtag)
  }
  newPageEtags = make(map[string]string)
  err := pageListCall.Pages(nil, func(pList *blogger.PageList) error {
    newListEtag = pList.Etag
    for _, p := range pList.Items {
      newPageEtags[p.Id] = p.Etag
    }
    return nil
  })
  if err != nil && ! googleapi.IsNotModified(err) {
    panic(err)
  }
  return
}

func(b *blogService) blogPost(blogId string, postId string) *blogger.Post {
  s := (*blogger.Service)(b)
  post, err := s.Posts.Get(blogId, postId).FetchImages(true).Do()
  if err != nil { panic(err) }
  return post
}

func(b *blogService) blogPage(blogId string, pageId string) *blogger.Page {
  s := (*blogger.Service)(b)
  page, err := s.Pages.Get(blogId, pageId).Do()
  if err != nil { panic(err) }
  return page
}

func(b *blogService) blog(blogId string) *blogger.Blog {
  s := (*blogger.Service)(b)
  blog, err := s.Blogs.Get(blogId).Do()
  if err != nil { panic(err) }
  return blog
}

func DefaultBloggerClient (ctx context.Context) (*http.Client, error) {
  return google.DefaultClient(ctx, blogger.BloggerReadonlyScope)
}

func BloggerClient(ctx context.Context, credentials []byte) (*http.Client, error) {
  jwtConf, err := google.JWTConfigFromJSON(credentials, blogger.BloggerReadonlyScope)
  if err != nil { return nil, err }
  return jwtConf.Client(ctx), nil
}

func SyncBlogger(ctx context.Context, blogId string, stasher BloggerStasher, client *http.Client) (err error) {

  defer func() {
    if r := recover(); r != nil {
      var ok bool
      err, ok = r.(error)
      if !ok {
        err = fmt.Errorf("pkg: %v", r)
      }
    }
  }()

  var b *blogService
  if bService, err := blogger.New(client); err != nil {
    return err
  } else {
    b = (*blogService)(bService)
  }
  syncBlog(ctx, blogId, stasher, b)
  syncBlogPosts(ctx, blogId, stasher, b)
  syncBlogPages(ctx, blogId, stasher, b)
  return nil
}

func syncBlog(ctx context.Context, blogId string, stasher BlogStasher, getter blogGetter) {
  newBlog := getter.blog(blogId)
  stasher.StashBlog(ctx, newBlog)
}

func syncBlogPosts(ctx context.Context, blogId string, stasher BlogPostStasher, getter blogPostGetter) {
  oldListEtag := stasher.GetPostListEtag(ctx, blogId)
  newListEtag, newPostEtags := getter.blogPostEtags(blogId, oldListEtag)
  if newListEtag == "" { return }
  newUpdated := time.Now()
  for id, etag := range newPostEtags {
    if stasher.HasPost(ctx, id, etag) {continue}
    post := getter.blogPost(blogId, id)
    stasher.StashPost(ctx, post)
  }
  stasher.StashPostEtags(ctx, blogId, newListEtag, newPostEtags, &newUpdated)
}

func syncBlogPages(ctx context.Context, blogId string, stasher BlogPageStasher, getter blogPageGetter) {
  oldListEtag := stasher.GetPageListEtag(ctx, blogId)
  newListEtag, newPageEtags := getter.blogPageEtags(blogId, oldListEtag)
  if newListEtag == "" { return }
  newUpdated := time.Now()
  for id, etag := range newPageEtags {
    if stasher.HasPage(ctx, id, etag) {continue}
    page := getter.blogPage(blogId, id)
    stasher.StashPage(ctx, page)
  }
  stasher.StashPageEtags(ctx, blogId, newListEtag, newPageEtags, &newUpdated)
}

func(m *MgoBlogStasher) GetPageListEtag(ctx context.Context, blogId string) (etag *string) {
  dbBlog := new(MgoBlog)
  err := m.BlogCollection.Find(bson.M{"blog.id": blogId}).Select(bson.M{"pageListEtag": 1}).One(dbBlog)
  if err != nil {
    if err == mgo.ErrNotFound {
      return nil
    }
    panic(err)
  }
  return dbBlog.PageListEtag
}
func(m *MgoBlogStasher) HasPage(ctx context.Context, pageId string, etag string) bool {
  n, err := m.BlogPageCollection.Find(bson.M{"blogPage.id": pageId, "blogPage.etag": etag}).Count()
  if err != nil { panic(err) }
  if n > 0 { return true }
  return false
}
func mgoWrapBlogPage(page *blogger.Page) *MgoBlogPage {
  dbPage := &MgoBlogPage{
    BlogPage: page,
  }
  pageUpdated, err := time.Parse(time.RFC3339, page.Updated)
  if err != nil { panic(err) }
  dbPage.Updated = &pageUpdated
  return dbPage
}
func(m *MgoBlogStasher) StashPage(ctx context.Context, page *blogger.Page) {
  dbPage := mgoWrapBlogPage(page)
  _, err := m.BlogPageCollection.Upsert(bson.M{"blogPage.id": page.Id, "updated": bson.M{"$lt": dbPage.Updated}}, &dbPage);
  if err != nil { panic(err) }
}
func(m *MgoBlogStasher) StashPageEtags(ctx context.Context, blogId string, listEtag string, pageEtags map[string]string, newUpdated *time.Time) {
  iter := m.BlogPageCollection.Find(nil).Select(bson.M{"dbPage.id": 1, "dbPage.etag": 1}).Select(bson.M{"_id": 1}).Iter()
  defer iter.Close()
  dbPage := new(MgoBlogPage)
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

  _, err := m.BlogCollection.Upsert(bson.M{"blog.id": blogId, "pageListUpdated": bson.M{"$lt": newUpdated}}, bson.M{"$set": &MgoBlog{PageListEtag: &listEtag, PageListUpdated: newUpdated}})
  if (err != nil) { panic(err) }
}

func(m *MgoBlogStasher) GetPostListEtag(ctx context.Context, blogId string) (etag *string) {
  dbBlog := new(MgoBlog)
  err := m.BlogCollection.Find(bson.M{"blog.id": blogId}).Select(bson.M{"postListEtag": 1}).One(dbBlog)
  if err != nil {
    if err == mgo.ErrNotFound {
      return nil
    }
    panic(err)
  }
  return dbBlog.PostListEtag
}
func(m *MgoBlogStasher) HasPost(ctx context.Context, postId string, etag string) bool {
  n, err := m.BlogPostCollection.Find(bson.M{"blogPost.id": postId, "blogPost.etag": etag}).Count()
  if err != nil { panic(err) }
  if n > 0 { return true }
  return false
}

func mgoWrapBlogPost(post *blogger.Post) *MgoBlogPost {
  dbPost := &MgoBlogPost{
    BlogPost: post,
  }
  postUpdated, err := time.Parse(time.RFC3339, post.Updated)
  if err != nil { panic(err) }
  dbPost.Updated = &postUpdated
  return dbPost
}
func(m *MgoBlogStasher) StashPost(ctx context.Context, post *blogger.Post) {
  dbPost := mgoWrapBlogPost(post)
  _, err := m.BlogPostCollection.Upsert(bson.M{"blogPost.id": post.Id, "updated": bson.M{"$lt": dbPost.Updated}}, dbPost);
  if err != nil { panic(err) }
}
func(m *MgoBlogStasher) StashPostEtags(ctx context.Context, blogId string, listEtag string, postEtags map[string]string, newUpdated *time.Time) {
  iter := m.BlogPostCollection.Find(nil).Select(bson.M{"dbPost.id": 1, "dbPost.etag": 1}).Select(bson.M{"_id": 1}).Iter()
  defer iter.Close()
  dbPost := new(MgoBlogPost)
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

  _, err := m.BlogCollection.Upsert(bson.M{"blog.id": blogId, "postListUpdated": bson.M{"$lt": newUpdated}}, bson.M{"$set": &MgoBlog{PostListEtag: &listEtag, PostListUpdated: newUpdated}})
  if (err != nil) { panic(err) }
}

func(m *MgoBlogStasher) StashBlog(ctx context.Context, blog *blogger.Blog) {
  dbBlog := MgoBlog{
    Blog: blog,
  }
  var err error
  *dbBlog.Updated, err = time.Parse(time.RFC3339, blog.Updated)
  if err != nil { panic(err) }
  _, err = m.BlogCollection.Upsert(bson.M{"blog.id": blog.Id, "updated": bson.M{"$lt": dbBlog.Updated}}, bson.M{"$set": &dbBlog})
  if (err != nil) { panic(err) }
}
