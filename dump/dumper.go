package dump

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"sort"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

const (
	darwinUserDataDir  = "Library/Application Support/Google/Chrome"
	linuxUserDataDir   = ".config/google-chrome"
	windowsUserDataDir = `Google\Chrome\User Data`
)

func getUserDataDir() string {
	var (
		userDataDir string
		home        string
	)

	switch runtime.GOOS {
	case "windows":
		home, _ = os.LookupEnv("LOCALAPPDATA")
		userDataDir = fmt.Sprintf("%s\\%s", home, windowsUserDataDir)
	case "linux":
		home, _ = os.LookupEnv("HOME")
		userDataDir = fmt.Sprintf("%s/%s", home, linuxUserDataDir)
		break
	case "darwin":
		home, _ = os.LookupEnv("HOME")
		userDataDir = fmt.Sprintf("%s/%s", home, darwinUserDataDir)
		break
	}
	return userDataDir
}

// ByDomain is a custom sort function
type ByDomain []*network.Cookie

func (a ByDomain) Len() int { return len(a) }
func (a ByDomain) Less(i, j int) bool {
	return a[i].Domain < a[j].Domain
}
func (a ByDomain) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func toMap(cookies []*network.Cookie) map[string][]*network.Cookie {
	var result map[string][]*network.Cookie = make(map[string][]*network.Cookie)
	for _, cookie := range cookies {
		_, ok := result[cookie.Domain]
		if ok {
			result[cookie.Domain] = append(result[cookie.Domain], cookie)
		} else {
			result[cookie.Domain] = []*network.Cookie{cookie}
		}
	}
	return result
}

// Dump Google Chrome's cookies
func Dump() {
	dir := getUserDataDir()
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		chromedp.UserDataDir(dir),
	)
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()
	task := chromedp.Tasks{
		chromedp.ActionFunc(func(ctx context.Context) error {
			var pretty bytes.Buffer
			cookies, err := network.GetAllCookies().Do(ctx)
			if err != nil {
				return err
			}
			sort.Sort(ByDomain(cookies))
			mapped := toMap(cookies)
			jsonData, err := json.Marshal(mapped)
			if err != nil {
				return err
			}
			err = json.Indent(&pretty, jsonData, "", "\t")
			if err != nil {
				return err
			}
			fmt.Println(pretty.String())
			return err
		}),
	}
	err := chromedp.Run(taskCtx, task)
	if err != nil {
		log.Fatal(err)
	}
}
