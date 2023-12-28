# Web Crawler
The Web Crawler is a command-line application written in Go that enables users to crawl a website and retrieve a list of links up to a specified depth. This documentation provides information about the application and its usage.

## Usage
To use the Web Crawler, execute the following command in your terminal:
```bash
go run main.go [flags]
```

### Flags
The application supports the following flags:
- `-url`: The URL of the website to crawl. Default is "https://duckduckgo.com".
- `-depth`: The maximum number of links deep to traverse. Default is 1.
- `-sameSite`: Check if links are under the same site. Default is true.
- `-loadDynamicContent`: Load dynamic content on the webpage. This may slow down the execution speed. Default is false.

### Example Usage
```bash
go run main.go -url=https://example.com -depth=2 -sameSite=true -loadDynamicContent=false
```

Or, execute twith the build:
```bash
go build -o webcrawler main.go

./webcrawler -url=https://example.com -depth=2 -sameSite=true -loadDynamicContent=false
```

### Output
The application prints a list of crawled links to the console, with each link prefixed by its corresponding index.

```bash
Links
----------------------------------------
001. https://example.com/page1
002. https://example.com/page2
003. https://example.com/page3

```

# Loading dynamic content
The flag `-loadDynamicContent` can be used if you need to load the dynamic content of the webpage. Most webpages has the content rendered by javascript after the initial HTML is rendered. The content rendered by javascript can have additional links in the webpage.

Thus, we use **[Chrome Devtools Protocol](https://chromedevtools.github.io/devtools-protocol/)** to load the webpage completely along with the javascript files and then analyse the links present in the webpage. Use this feature only if required. This may slow down the execution.

### Notes
- Ensure you have Go installed on your machine.
- Modify the flags as needed to customize the crawling behavior.

