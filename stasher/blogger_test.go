package stasher

import (
  "testing"
  "context"
  "google.golang.org/api/blogger/v3"
  "gopkg.in/mgo.v2/bson"
)

type testBlogGetter blogger.Blog

func (b *testBlogGetter) blog(blogId string) *blogger.Blog {
  return (*blogger.Blog)(b)
}

type testBlogStasher bson.M

func (b *testBlogStasher) StashBlog(ctx context.Context, blog *blogger.Blog) {
  inBlog := &MgoBlog{Blog: blog}
  bytes, err := bson.Marshal(inBlog)
  if err != nil { panic(err) }
  err = bson.Unmarshal(bytes, (*bson.M)(b))
  if err != nil { panic(err) }
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
