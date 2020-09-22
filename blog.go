package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// AUTHOR : name of the post's author
const AUTHOR string = "Aaron Goidel"

// PATH : path to blog posts
const PATH string = "/Users/agoidel/Documents/development/this-is-me/pages/blog/posts"

type blogT struct {
	title     string
	author    string
	slug      string
	timeStamp time.Time
	wordCount int
	lines     []string
}

func parseRawMd(file *os.File, blog *blogT) {
	numWords := 0
	var isCode bool

	scanner := bufio.NewScanner(file)
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

	var post blogT

	post.author = AUTHOR

	inFile, err := os.Open(*fileName)
	if err != nil {
		panic(err)
	}
	defer inFile.Close()

	parseRawMd(inFile, &post)

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

	post.timeStamp = time.Now()

	outFile, err := os.Create(PATH + "/" + post.slug + ".md")
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
