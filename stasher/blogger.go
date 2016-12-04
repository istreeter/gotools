package stasher

import (
  "context"
  "time"
  "net/http"
  "fmt"
  "google.golang.org/api/blogger/v3"
  "google.golang.org/api/googleapi"
  "golang.org/x/oauth2/google"
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
  if err != nil { panic(fmt.Sprintf("Error getting blog %v: %v", blogId, err.Error())) }
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

