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

	"github.com/go-git/go-git"
	"github.com/go-git/go-git/plumbing/object"
)

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
	Email    string `json:"email"`
	ProjPath string `json:"projPath"`
	PostPath string `json:"postSubDir"`
}

var config configurationT

func log(tag string, msg string) {
	out := fmt.Sprintf("[%s]: %s", tag, msg)
	fmt.Println(out)
}

func handleErr(err error) {
	if err != nil {
		panic(err)
	}
}

func parseRawMd(path string, blog *blogT) {
	log("INFO", "begin parsing markdown")
	inFile, err := os.Open(path)
	handleErr(err)
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
	log("INFO", "finished parsing raw blog material")
	log("INFO", fmt.Sprintf("final word count: %d", numWords))
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
	log("INFO", "generating slug")
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
	postPath, _ := filepath.EvalSymlinks(config.ProjPath + "/" + config.PostPath)
	outFile, err := os.Create(postPath + "/" + post.slug + ".md")
	handleErr(err)
	defer outFile.Close()

	log("INFO", "generating front matter")
	frontMatter := []string{
		"---",
		fmt.Sprintf("title: \"%s\"", post.title),
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

	log("INFO", "writing data to file")
	for _, line := range post.lines {
		fmt.Fprintln(outFile, line)
	}
	outFile.Close()
	log("INFO", "done generating post")
}

func deployPost(post blogT) {
	log("INFO", "deploying post")

	repoPath, _ := filepath.EvalSymlinks(config.ProjPath)
	repo, err := git.PlainOpen(repoPath)
	handleErr(err)

	wTree, err := repo.Worktree()

	postfile := config.PostPath + "/" + post.slug + ".md"
	_, err = wTree.Add(postfile)
	handleErr(err)
	log("GIT", "added file "+postfile)

	log("GIT", "checking status")
	status, err := wTree.Status()
	handleErr(err)
	fmt.Println(status)

	commit, err := wTree.Commit("feat(blog): add new post", &git.CommitOptions{
		Author: &object.Signature{
			Name:  config.Author,
			Email: config.Email,
			When:  time.Now(),
		},
	})
	handleErr(err)

	log("GIT", "committing post")
	obj, err := repo.CommitObject(commit)
	handleErr(err)

	fmt.Println(obj)

	log("GIT", "pushing to remote repository")
	err = repo.Push(&git.PushOptions{RemoteName: "origin"})
	handleErr(err)
}

func getCleanInput(prompt string) string {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print(prompt)
	raw, _ := reader.ReadString('\n')

	return strings.TrimSuffix(raw, "\n")
}

func runSetup() {
	log("SETUP", "starting setup routine")
	home := os.Getenv("HOME") + "/.blog"
	configPath := home + "/.config.json"

	if _, err := os.Stat(configPath); err == nil {
		log("SETUP", "found config file")
		file, _ := os.Open(configPath)
		defer file.Close()
		decoder := json.NewDecoder(file)
		err := decoder.Decode(&config)
		handleErr(err)
	} else if os.IsNotExist(err) {
		log("SETUP", "no config file found")
		os.Mkdir(home, 0755)
		configFile, err := os.Create(configPath)
		handleErr(err)
		log("SETUP", "created new config file at: "+configPath)

		encoder := json.NewEncoder(configFile)

		config.Author = getCleanInput("Enter your name: ")
		config.Email = getCleanInput("Enter your email: ")
		config.ProjPath = getCleanInput("Path to git repo for blog: ")
		subdirPrompt := fmt.Sprintf("Subdirectory containing blog posts: %s", config.ProjPath)
		config.PostPath = getCleanInput(subdirPrompt)

		encoder.Encode(config)
		log("SETUP", "wrote preferences to config")

		configFile.Close()
	}
	log("SETUP", "done setup!")
}

func main() {
	fileName := flag.String("f", "", "Path to markdown file. (Required)")
	push := flag.Bool("push", false, "Push new blog post to master.")

	flag.Parse()

	if *fileName == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	runSetup()

	var post blogT

	post.author = config.Author

	parseRawMd(*fileName, &post)

	getUserInputtedFields(&post)

	post.timeStamp = time.Now()

	writePost(&post)

	if *push {
		deployPost(post)
	}
}
