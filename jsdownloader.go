package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
)

func exists(url string, all_urls []string) bool {
    for _, v := range all_urls {
        if v == url {
            return true
        }
    }
    return false
}

func format_match(full string, rel string) string {
    u, err1 := url.Parse(full)
    r, err2 := url.Parse(rel)
    if err1 != nil || err2 != nil {
        fmt.Println("couldn't resolve reference: ", err1, err2)
        return ""
    }
    resolved := u.ResolveReference(r)
    return resolved.String()
}

func format_matches(url string, matches *[]string) {
    for i, match := range *matches {
        (*matches)[i] = format_match(url, match)
    }
}

func any_urls_are_the_same(data []string) {
    seen := make(map[string]bool)
    for _, str := range data {
        if seen[str] {
            panic("found duplicate")
        }
        seen[str] = true
    }
}

func get_js_urls(url string) []string {
    resp, err := http.Get(url)
    if err != nil {
        fmt.Println(err)
        return []string{}
    }
		defer resp.Body.Close()
    bodyBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        log.Printf("Error reading response body: %v", err)
        return []string{}
    }
    body := string(bodyBytes)
    // pattern := `"((https?:\/\/[^"\n\r]+\.js|[^"\n\r]+\.js))"`
    // pattern := `"(https?:\/\/[^"\n\r]+\.js|[^"\n\r]+\.js)"`
    // pattern := `"((https?:\/\/[^"\s]+?\.js|[^"\s]+?\.js))"`
    // pattern := `\b"https?:\/\/[^"\s]+?\.js(?:\?[^"\s]*)?"|\b"[^"\s]+?\.js(?:\?[^"\s]*)?"`
    pattern := `"((https?:\/\/[^"\s]+?\.js(?:\?[^"\s]*)?|[^"\s]+?\.js(?:\?[^"\s]*)?))"`
    re := regexp.MustCompile(pattern)
    var matches []string
    raw_matches := re.FindAllStringSubmatch(body, -1)
    for _, match := range raw_matches {
        matches = append(matches, match[1])
    }
    format_matches(url, &matches)
    return matches
}

func get_all_js_urls(all_urls []string, previous_fresh_urls []string) []string {
    any_urls_are_the_same(previous_fresh_urls)
    if len(previous_fresh_urls) == 0 { return previous_fresh_urls }
    var next_fresh_urls []string 
    for _, url := range previous_fresh_urls {
        newUrls := get_js_urls(url)
        for _, url := range newUrls {
            if len(url) > 0 && !exists(url, all_urls) {
                next_fresh_urls = append(next_fresh_urls, url)
                all_urls = append(all_urls, url)
            }
        }
    }
		return append(previous_fresh_urls, get_all_js_urls(all_urls, next_fresh_urls)...)
}

func printall(name string, list []string) {
	fmt.Println(name, ": ")
	for _, name := range list {
		fmt.Println("  - ", name)
	}
}

func download(urlStr string) error {
		fmt.Println("Downloading ", urlStr)
		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			return err
		}
		domain := parsedURL.Host
		path := filepath.Join(domain, parsedURL.Path)

		// Handle query parameters by appending them to the filename
		if parsedURL.RawQuery != "" {
			extension := filepath.Ext(path)
			pathWithoutExtension := path[0 : len(path)-len(extension)]
			path = pathWithoutExtension + "-" + parsedURL.RawQuery + extension
		}


		// TODO: figure out another way of doing this without 
		// including all 10,000 possible TLDs?
		extention_list := []string{".net", ".info", ".com", ".xyz", ".io", ".soy"}
		if slices.Contains(extention_list, filepath.Ext(path)) {
			path = filepath.Join(path, "index.html")
		}

		resp, err := http.Get(urlStr)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		dir := filepath.Dir(path)

		// Create directory structure
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}

		// Create the file
		out, err := os.Create(path)
		if err != nil {
			return err
		}
		defer out.Close()

		// Write the body to file
		_, err = io.Copy(out, resp.Body)
		return err
}

func main() {
		flag.Usage = func() {
			fmt.Fprintf(os.Stderr, "Usage: %s [options] <url>\n\n", os.Args[0])
			fmt.Fprintf(os.Stderr, "Options:\n")
			fmt.Fprintf(os.Stderr, "--download            Download all files\n\n")
		}

		downloadFlag := flag.Bool("download", false, "Download all files")

		flag.Parse()
		args := flag.Args()

		if len(args) != 1 {
			flag.Usage()
			os.Exit(1)
		}

		url := []string{args[0]}

    final_urls := get_all_js_urls(url[:], url)

		printall("final urls", final_urls)

		if *downloadFlag {
			for _, url := range final_urls {
				err := download(url)
				if err != nil {
					fmt.Println(err)
				}
			}
		}

}
