package rss

import (
	"github.com/takeoutfm/takeout/lib/client"
	//"github.com/takeoutfm/takeout/lib/date"
	"testing"
)

func TestRSS(t *testing.T) {
	urls := []string{
		// "https://feeds.twit.tv/twit.xml",
		// "https://feeds.twit.tv/sn.xml",
		// "https://www.pbs.org/newshour/feeds/rss/podcasts/show",
		"http://feeds.feedburner.com/TEDTalks_audio",
	}

	var config client.Config
	config.UserAgent = "rss/test"
	rss := NewRSS(client.NewGetter(config))
	for i := 0; i < len(urls); i++ {
		url := urls[i]
		feed, err := rss.Fetch(url)
		if err != nil {
			t.Logf("%v\n", err)
		}
		t.Logf("%s [%s]\n", feed.Title, feed.Link())
		//t.Logf("%s\n", channel.LastBuildTime())
		for _, e := range feed.Items {
			//t.Logf("%s - %s\n", date.Format(e.PublishTime), e.Title)
			//t.Logf("%s %d %s\n", e.ContentType, e.Size, e.URL)
			t.Logf("%s %s\n", e.Title, e.ItemImage())
		}
	}
}
