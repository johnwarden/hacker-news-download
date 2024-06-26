package main

import (
  "context"
  "encoding/json"
  "fmt"
  "os"
  "os/signal"
  "syscall"

  "github.com/johnwarden/hn"
  "github.com/pkg/errors"
  "github.com/schollz/progressbar/v3"
  "github.com/jessevdk/go-flags"
)

const maxGoroutines = 20

func getItemWithComments(ctx context.Context, id int) {

  client := hn.DefaultClient

  var out chan hn.Item
  out = make(chan hn.Item, 1)


  item, err := client.Item(ctx, id)
  if err != nil {
    fmt.Fprintln(os.Stderr, errors.Wrapf(err, "getComments for item %d", id))
  }

  n := item.Descendants
  if n == 0 && len(item.Kids) != 0 {
    n = -1
    fmt.Fprintln(os.Stderr, fmt.Sprintf("Fetching unknown number of descendants"))
  } else {
    fmt.Fprintln(os.Stderr, fmt.Sprintf("Fetching %d descendants", n))    
  }
  bar := progressbar.Default(int64(n))
  out <- *item

  go func() {

    err = getComments(ctx, client, item.Kids, out)
    if err != nil {
      fmt.Fprintln(os.Stderr, errors.Wrapf(err, "getComments for item %d", id))
    }
    close(out)
  }()


  for item := range out {
    bar.Add(1)
    // fmt.Println("Got an item")
    if item.ID == 0 {
      continue
    }
    jsonText, err := json.Marshal(item)
    if err != nil {
      fmt.Fprintln(os.Stderr, errors.Wrap(err, "Marshalling json"))
      continue
    }

    fmt.Println(string(jsonText))
  }


}

func getComments(ctx context.Context, client *hn.Client, ids []int, out chan hn.Item) error {
  if len(ids) == 0 {
    return nil
  }
  // fmt.Println("Getting comments", ids)
  comments, err := client.GetItems(ctx, ids, maxGoroutines)
  if err != nil {
    return errors.Wrapf(err, "client.GetItems(%v)", ids)
  }

  for _, comment := range comments {
    // fmt.Printf("Outputting comments %#v\n", comment)
    out <- comment

    err := getComments(ctx, client, comment.Kids, out)
    if err != nil {
      return errors.Wrapf(err, "getComments for item %d", comment.ID)
    }
  }

  return nil
}


var opts struct {
  // Slice of bool will append 'true' each time the option
  StoryID int `short:"s" long:"storyID" description:"ID of item to download" required:"true"`
}


func withCancelOnInterrupt(ctx context.Context) (context.Context, func())  {

  // trap Ctrl+C and call cancel on the context
  ctx, cancel := context.WithCancel(ctx)
  c := make(chan os.Signal, 1)
  signal.Notify(c, os.Interrupt, syscall.SIGTERM)
  go func() {
    select {
    case <-c:
      cancel()
    case <-ctx.Done():
    }
  }()


  return ctx, func() {
    signal.Stop(c)
    cancel()
  }
}

func main() {

  _, err := flags.ParseArgs(&opts, os.Args)

  if err != nil {
      fmt.Fprintln(os.Stderr, err)
      os.Exit(1)
  }

  ctx := context.Background()
  ctx, cancel := withCancelOnInterrupt(ctx)
  defer cancel()

  fmt.Fprintln(os.Stderr, fmt.Sprintf("Fetching item %d", opts.StoryID))

  getItemWithComments(ctx, opts.StoryID)

}
