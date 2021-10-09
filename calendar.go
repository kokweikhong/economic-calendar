package calendar

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
)

const (
	url string = "https://www.investing.com/economic-calendar/"
)

type DateConfig struct {
	Start, End time.Time
}

type economic struct {
	Time      time.Time
	Currency  string
	Important string
	Event     string
	Actual    string
	Forecast  string
	Previous  string
}

func GetEconomicCalendar(date *DateConfig) []*economic {
	options := []chromedp.ExecAllocatorOption{
		chromedp.Flag("headless", false),
		chromedp.Flag("hide-scrollbars", false),
		chromedp.Flag("mute-audio", false),
		chromedp.UserAgent(`Mozilla/5.0 (Windows NT 6.3; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.103 Safari/537.36`),
		chromedp.Flag("blink-settings", "imagesEnabled=false"),
	}
	options = append(chromedp.DefaultExecAllocatorOptions[:], options...)
	allocatorCtx, cancel := chromedp.NewExecAllocator(context.Background(), options...)
	defer cancel()
	// create context
	ctx, cancel := chromedp.NewContext(allocatorCtx)
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	var nodeTasks chromedp.Tasks
	var nodes []*cdp.Node
	if date != nil {
		start, end := date.dateValidation()
		nodeTasks = append(nodeTasks, selectDate(start, end))
	}
	nodeTasks = append(nodeTasks, retrieveNodes(&nodes))
	fmt.Println(len(nodeTasks))
	chromedp.Run(ctx,
		navigateTask(),
		nodeTasks,
	)
	var result []*economic
	wg := &sync.WaitGroup{}
	mu := &sync.Mutex{}
	wg.Add(len(nodes))
	for i := 0; i < len(nodes); i++ {
		go func(i int) {
			chromedpOpts := []func(*chromedp.Selector){
				chromedp.ByQuery,
				chromedp.FromNode(nodes[i]),
			}
			eco := &economic{}
			datetime, _ := nodes[i].Attribute("data-event-datetime")
			var checkImportant bool
			chromedp.Run(ctx,
				chromedp.Text("td:nth-child(2)", &eco.Currency, chromedpOpts...),
				chromedp.AttributeValue("td:nth-child(3)", "title", &eco.Important, &checkImportant,
					chromedpOpts...),
				chromedp.Text("td:nth-child(4)", &eco.Event, chromedpOpts...),
				chromedp.Text("td:nth-child(5)", &eco.Actual, chromedpOpts...),
				chromedp.Text("td:nth-child(6)", &eco.Forecast, chromedpOpts...),
				chromedp.Text("td:nth-child(7)", &eco.Previous, chromedpOpts...),
			)
			eco.Time, _ = time.Parse("2006/01/02 15:04:05", datetime)
			splitImportant := strings.Split(eco.Important, " ")
			if len(splitImportant) > 1 {
				eco.Important = splitImportant[0]
			}
			mu.Lock()
			result = append(result, eco)
			fmt.Println(eco)
			wg.Done()
			mu.Unlock()
		}(i)
	}
	wg.Wait()
	return result
}

func (dc *DateConfig) dateValidation() (string, string) {
	var start, end string
	if dc.End.Before(dc.Start) {
		start = time.Now().AddDate(0, 0, -3).Format("01/02/2006")
		end = time.Now().AddDate(0, 0, 3).Format("01/02/2006")
		return start, end
	}
	return dc.Start.Format("01/02/2006"), dc.End.Format("01/02/2006")
}

func navigateTask() chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.Navigate(url),
		chromedp.WaitVisible("div#economicCurrentTime"),
		chromedp.Click("div#economicCurrentTime"),
		chromedp.WaitVisible("li#liTz55"),
		chromedp.Click("li#liTz55"),
		chromedp.WaitVisible("table#economicCalendarData"),
	}
}

func selectDate(start, end string) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.Click("a#datePickerToggleBtn"),
		chromedp.Focus("input#startDate"),
		chromedp.SetValue("input#startDate", start),
		chromedp.Focus("input#endDate"),
		chromedp.SetValue("input#endDate", end),
		chromedp.SendKeys("input#endDate", " "),
		chromedp.SendKeys("input#endDate", kb.Backspace),
		chromedp.Click("a#applyBtn"),
		chromedp.WaitVisible("table#economicCalendarData"),
		chromedp.Sleep(3 * time.Second),
	}
}

func retrieveNodes(nodes *[]*cdp.Node) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.Nodes("tr.js-event-item", nodes, chromedp.ByQueryAll),
	}
}
