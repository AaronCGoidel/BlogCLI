package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const author string = "Aaron Goidel"
const postDirPath string = "/Users/agoidel/Documents/development/this-is-me/pages/blog/posts"

type blogT struct {
	title     string
	author    string
	slug      string
	timeStamp time.Time
	wordCount int
	lines     []string
}

type configurationT struct {
	Author   string `json:"author"`
	PostPath string `json:"postPath"`
}

var config configurationT

func parseRawMd(path string, blog *blogT) {
	inFile, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer inFile.Close()

	numWords := 0
	var isCode bool

	scanner := bufio.NewScanner(inFile)
	for scanner.Scan() {
		line := scanner.Text()
		blog.lines = append(blog.lines, line)

		// skip code blocks
		r := regexp.MustCompile("(```.*)")
		if r.MatchString(line) {
			isCode = !isCode
			continue
		}
		if isCode {
			continue
		}

		// get rid of header, quote, and enum specifiers
		r = regexp.MustCompile("((#+|>|[0-9]+.)\\s)")
		line = r.ReplaceAllString(line, "")

		// get rid of images
		r = regexp.MustCompile("!\\[[^\\]]*\\]\\([^)]*\\)")
		line = r.ReplaceAllString(line, "")

		// get rid of html tags
		r = regexp.MustCompile("</?[^>]*>")
		line = r.ReplaceAllString(line, "")

		// get rid of links
		r = regexp.MustCompile("(\\(https?:\\/\\/.*\\))")
		line = r.ReplaceAllString(line, "")

		numWords += len(strings.Fields(line))

	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}
	blog.wordCount = numWords
}

func toTitle(str string) string {
	words := strings.Fields(str)
	smallwords := " a an on the to " // words to keep lowercase

	for i, word := range words {
		// if the word is small and not sentence initial keep it lowercase
		if i != 0 && strings.Contains(smallwords, " "+word+" ") {
			words[i] = word
		} else {
			words[i] = strings.Title(word)
		}
	}
	return strings.Join(words, " ")
}

func slugify(str string) string {
	slug := strings.ToLower(str)
	r := regexp.MustCompile("[^\\w ]+")
	slug = r.ReplaceAllString(slug, "")
	r = regexp.MustCompile("\\ +")
	slug = r.ReplaceAllString(slug, "-")

	return slug
}

func getUserInputtedFields(post *blogT) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter title: ")
	rawTitle, _ := reader.ReadString('\n')
	post.title = toTitle(rawTitle)

	fmt.Print("Use default slug (Y/n): ")
	res, _ := reader.ReadString('\n')
	useDefaultSlug := res != "n"

	if !useDefaultSlug {
		fmt.Print("Enter slug: ")
		post.slug, _ = reader.ReadString('\n')
	} else {
		post.slug = slugify(rawTitle)
	}
}

func writePost(post *blogT) {
	postPath, _ := filepath.EvalSymlinks(config.PostPath)
	outFile, err := os.Create(postPath + "/" + post.slug + ".md")
	if err != nil {
		panic(err)
	}
	defer outFile.Close()

	frontMatter := []string{
		"---",
		"title: " + post.title,
		"author: " + post.author,
		"slug: " + post.slug,
		fmt.Sprintf("date: \"%s\"", post.timeStamp.Format("2006-01-02 15:04:05")),
		fmt.Sprintf("wcount: %d", post.wordCount),
		"---",
	}

	for _, line := range frontMatter {
		fmt.Fprintln(outFile, line)
	}
	fmt.Fprintln(outFile, "")

	for _, line := range post.lines {
		fmt.Fprintln(outFile, line)
	}
	outFile.Close()
}

func getCleanInput(prompt string) string {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print(prompt)
	raw, _ := reader.ReadString('\n')

	return strings.TrimSuffix(raw, "\n")
}

func runSetup() {
	home := os.Getenv("HOME") + "/.blog"
	configPath := home + "/.config.json"

	if _, err := os.Stat(configPath); err == nil {
		file, _ := os.Open(configPath)
		defer file.Close()
		decoder := json.NewDecoder(file)
		err := decoder.Decode(&config)
		if err != nil {
			fmt.Println("error:", err)
		}
	} else if os.IsNotExist(err) {
		os.Mkdir(home, 0755)
		configFile, err := os.Create(configPath)
		if err != nil {
			panic(err)
		}

		encoder := json.NewEncoder(configFile)

		config.Author = getCleanInput("Enter your name: ")
		config.PostPath = getCleanInput("Path to blog posts: ")

		encoder.Encode(config)

		configFile.Close()
	}
}

func main() {
	fileName := flag.String("f", "", "Path to markdown file. (Required)")
	push := flag.Bool("push", false, "Push new blog post to master.")

	flag.Parse()

	if *fileName == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if !*push {
	}

	runSetup()

	var post blogT

	post.author = config.Author

	parseRawMd(*fileName, &post)

	getUserInputtedFields(&post)

	post.timeStamp = time.Now()

	writePost(&post)
}
