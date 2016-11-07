package stasher

import (
  "context"
  "time"
  "google.golang.org/api/blogger/v3"
  "google.golang.org/api/googleapi"
  "gopkg.in/mgo.v2"
  "gopkg.in/mgo.v2/bson"
)

const DefaultBlogPageCollection = "blog_page"
const DefaultBlogPostCollection = "blog_post"
const DefaultBlogCollection = "blog"

type BlogPageStasher interface {
  GetPageListEtag(ctx context.Context, blogId string) (etag *string)
  HasPage(ctx context.Context, pageId string, etag string) bool
  StashPage(context.Context, *blogger.Page)
  StashPageEtags(ctx context.Context, blogId string, listEtag string, pageEtags map[string]string, updated *time.Time)
}

type MgoBlogStasher struct {
  BlogPageCollection *mgo.Collection
  BlogPostCollection *mgo.Collection
  BlogCollection *mgo.Collection
}

type MgoBlogPage struct {
  Id *bson.ObjectId `bson:"_id,omitempty"`
  BlogPage *blogger.Page
  Updated *time.Time
}

type MgoBlog struct {
  Id *bson.ObjectId `bson:"_id,omitempty"`
  Blog *blogger.Blog `bson:"omitempty"`
  PostListEtag *string `bson:"omitempty"`
  PostListUpdated *time.Time `bson:"omitempty"`
  PageListEtag *string `bson:"omitempty"`
  PageListUpdated *time.Time `bson:"omitempty"`
}

func DefaultMgoStasher(db *mgo.Database) (m *MgoBlogStasher) {
  m = new(MgoBlogStasher)
  m.BlogPageCollection = db.C(DefaultBlogPageCollection)
  m.BlogPostCollection = db.C(DefaultBlogPostCollection)
  m.BlogCollection = db.C(DefaultBlogCollection)
  return
}

type blogPageGetter interface {
  blogPageEtags(blogId string, oldListEtag *string) (newListEtag string, newPageEtags map[string]string)
  blogPage(blogId string, pageId string) *blogger.Page
}

type blogService blogger.Service;

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

func(b *blogService) blogPage(blogId string, pageId string) *blogger.Page {
  s := (*blogger.Service)(b)
  page, err := s.Pages.Get(blogId, pageId).Do()
  if err != nil { panic(err) }
  return page
}


func syncBlogPages(ctx context.Context, blogId string, stasher BlogPageStasher, getter blogPageGetter) {
  oldListEtag := stasher.GetPageListEtag(ctx, blogId)
  newListEtag, newPageEtags := getter.blogPageEtags(blogId, oldListEtag)
  newUpdated := time.Now()
  if newListEtag == "" { return }
  for id, etag := range newPageEtags {
    if stasher.HasPage(ctx, id, etag) {continue}
    page := getter.blogPage(blogId, id)
    stasher.StashPage(ctx, page)
  }
  stasher.StashPageEtags(ctx, blogId, newListEtag, newPageEtags, &newUpdated)
}

func(m *MgoBlogStasher) GetPageListEtag(ctx context.Context, blogId string) (etag *string) {
  dbBlog := new(MgoBlog)
  err := m.BlogCollection.Find(bson.M{"blog.id": blogId}).Select(bson.M{"PageListEtag": 1}).One(dbBlog)
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
func(m *MgoBlogStasher) StashPage(ctx context.Context, page *blogger.Page) {
  dbPage := MgoBlogPage{
    BlogPage: page,
  }
  var err error
  *dbPage.Updated, err = time.Parse(page.Updated, time.RFC3339)
  if err != nil { panic(err) }
  _, err = m.BlogPageCollection.Upsert(bson.M{"blogPage.id": page.Id, "updated": bson.M{"$lt": dbPage.Updated}}, &dbPage);
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
