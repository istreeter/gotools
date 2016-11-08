package stasher

import (
  "testing"
  "context"
  "time"
  "google.golang.org/api/blogger/v3"
  "gopkg.in/mgo.v2/bson"
)

type testBlogGetter blogger.Blog

func (g *testBlogGetter) blog(blogId string) *blogger.Blog {
  return (*blogger.Blog)(g)
}

type testBlogPostGetter struct {
  listEtag string
  posts map[string]*blogger.Post
}

func(g *testBlogPostGetter)  blogPostEtags(blogId string, oldListEtag *string) (string, map[string]string) {
  if oldListEtag != nil && *oldListEtag == g.listEtag {
    return "", nil
  }
  newListEtag := g.listEtag
  newPostEtags := make(map[string]string)
  for postId, post := range g.posts {
    newPostEtags[postId] = post.Etag
  }
  return newListEtag, newPostEtags
}

func(g *testBlogPostGetter) blogPost(blogId string, postId string) *blogger.Post {
  return g.posts[postId]
}

type testBlogPageGetter struct {
  listEtag string
  pages map[string]*blogger.Page
}

func(g *testBlogPageGetter)  blogPageEtags(blogId string, oldListEtag *string) (string, map[string]string) {
  if oldListEtag != nil && *oldListEtag == g.listEtag {
    return "", nil
  }
  newListEtag := g.listEtag
  newPageEtags := make(map[string]string)
  for pageId, page := range g.pages {
    newPageEtags[pageId] = page.Etag
  }
  return newListEtag, newPageEtags
}

func(g *testBlogPageGetter) blogPage(blogId string, pageId string) *blogger.Page {
  return g.pages[pageId]
}

type testBlogStasher bson.M

func (b *testBlogStasher) StashBlog(ctx context.Context, blog *blogger.Blog) {
  inBlog := &MgoBlog{Blog: blog}
  bytes, err := bson.Marshal(inBlog)
  if err != nil { panic(err) }
  err = bson.Unmarshal(bytes, (*bson.M)(b))
  if err != nil { panic(err) }
}

type testBlogPostStasher struct {
  listEtag string
  updated *time.Time
  posts map[string]*MgoBlogPost
}
func(s *testBlogPostStasher) GetPostListEtag(ctx context.Context, blogId string) *string {
  if s.listEtag == "" { return nil }
  return &s.listEtag
}
func(s *testBlogPostStasher) HasPost(ctx context.Context, postId string, etag string) bool {
  post, ok := s.posts[postId]
  if !ok { return false }
  if post.BlogPost.Etag != etag { return false }
  return true
}
func(s *testBlogPostStasher) StashPost(ctx context.Context, post *blogger.Post) {
  s.posts[post.Id] = mgoWrapBlogPost(post)
}
func(s *testBlogPostStasher) StashPostEtags(ctx context.Context, blogId string, listEtag string, postEtags map[string]string, updated *time.Time) {
  s.listEtag = listEtag
  s.updated = updated
  for postId, post := range s.posts {
    if postEtags[postId] != post.BlogPost.Etag {
      delete(s.posts, postId)
    }
  }
}

type testBlogPageStasher struct {
  listEtag string
  updated *time.Time
  pages map[string]*MgoBlogPage
}
func(s *testBlogPageStasher) GetPageListEtag(ctx context.Context, blogId string) *string {
  if s.listEtag == "" { return nil }
  return &s.listEtag
}
func(s *testBlogPageStasher) HasPage(ctx context.Context, pageId string, etag string) bool {
  page, ok := s.pages[pageId]
  if !ok { return false }
  if page.BlogPage.Etag != etag { return false }
  return true
}
func(s *testBlogPageStasher) StashPage(ctx context.Context, page *blogger.Page) {
  s.pages[page.Id] = mgoWrapBlogPage(page)
}
func(s *testBlogPageStasher) StashPageEtags(ctx context.Context, blogId string, listEtag string, pageEtags map[string]string, updated *time.Time) {
  s.listEtag = listEtag
  s.updated = updated
  for pageId, page := range s.pages {
    if pageEtags[pageId] != page.BlogPage.Etag {
      delete(s.pages, pageId)
    }
  }
}


func TestSyncBlog(t *testing.T) {
  defer func() {
    if err := recover(); err != nil {
      t.Fatal(err)
    }
  }()

  blog := &blogger.Blog{Id: "10", Updated: "2016-11-08T06:10:34+00:00"}

  stasher := testBlogStasher{}

  getter := (*testBlogGetter)(blog)
  syncBlog(nil, "10", &stasher, getter)

  if outDbBlog, ok := stasher["blog"]; ok {
    if outBlog, ok2 := outDbBlog.(bson.M); ok2 {
      if blogId, ok3 := outBlog["id"]; ok3 {
        if blogId.(string) != blog.Id {
          t.Errorf("Ids do not match: %v %v", blog.Id, blogId)
        }
      } else {
        t.Errorf("blog is missing id")
      }
    } else {
      t.Errorf("blog is unexpected type")
    }
  } else {
    t.Errorf("stasher does not have a blog")
  }

  for key, _ := range stasher {
    if key != "blog" {
      t.Errorf("stasher should not have %v", key)
    }
  }
}

func TestSyncBlogPosts(t *testing.T) {
  defer func() {
    if err := recover(); err != nil {
      t.Fatal(err)
    }
  }()

  stasher := &testBlogPostStasher{}
  stasher.posts = make(map[string]*MgoBlogPost)
  stasher.listEtag = "OldListEtag"
  now := time.Now()
  stasher.updated = &now
  getter := &testBlogPostGetter{}
  getter.posts = make(map[string]*blogger.Post)
  getter.listEtag = "NewListEtag"

  p1 := &blogger.Post{Id: "1", Updated: "2016-11-08T06:10:34Z", Etag: "TAG1"}
  stasher.posts[p1.Id] = mgoWrapBlogPost(p1)
  getter.posts[p1.Id] = p1

  p2 := &blogger.Post{Id: "2", Updated: "2016-11-09T06:10:34Z", Etag: "TAG2"}
  getter.posts[p2.Id] = p2

  syncBlogPosts(nil, "10", stasher, getter)
  if l := len(stasher.posts); l != 2 {
      t.Errorf("number of posts: expected 2, got %v", l)
  }
  stasher.listEtag = "OldListEtag"

  p3 := &blogger.Post{Id: "3", Updated: "2016-11-10T06:10:34Z", Etag: "TAG3"}
  stasher.posts[p3.Id] = mgoWrapBlogPost(p3)
  syncBlogPosts(nil, "10", stasher, getter)
  if l := len(stasher.posts); l != 2 {
      t.Errorf("number of posts: expected 2, got %v", l)
  }
}

func TestSyncBlogPages(t *testing.T) {
  defer func() {
    if err := recover(); err != nil {
      t.Fatal(err)
    }
  }()

  stasher := &testBlogPageStasher{}
  stasher.pages = make(map[string]*MgoBlogPage)
  stasher.listEtag = "OldListEtag"
  now := time.Now()
  stasher.updated = &now
  getter := &testBlogPageGetter{}
  getter.pages = make(map[string]*blogger.Page)
  getter.listEtag = "NewListEtag"

  p1 := &blogger.Page{Id: "1", Updated: "2016-11-08T06:10:34Z", Etag: "TAG1"}
  stasher.pages[p1.Id] = mgoWrapBlogPage(p1)
  getter.pages[p1.Id] = p1

  p2 := &blogger.Page{Id: "2", Updated: "2016-11-09T06:10:34Z", Etag: "TAG2"}
  getter.pages[p2.Id] = p2

  syncBlogPages(nil, "10", stasher, getter)
  if l := len(stasher.pages); l != 2 {
      t.Errorf("number of pages: expected 2, got %v", l)
  }
  stasher.listEtag = "OldListEtag"

  p3 := &blogger.Page{Id: "3", Updated: "2016-11-10T06:10:34Z", Etag: "TAG3"}
  stasher.pages[p3.Id] = mgoWrapBlogPage(p3)
  syncBlogPages(nil, "10", stasher, getter)
  if l := len(stasher.pages); l != 2 {
      t.Errorf("number of pages: expected 2, got %v", l)
  }
}
