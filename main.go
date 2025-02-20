package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dinopy/blogaggregator/internal/config"
	"github.com/dinopy/blogaggregator/internal/database"
	"github.com/dinopy/blogaggregator/internal/parser"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type state struct {
	config *config.Config
	db	   *database.Queries
}

type command struct {
	name		string
	arguments	[]string
}

type commands struct {
	c		map[string]func(*state, command) error
}

func (c commands) handleLogin(s *state, cmd command) error {
	if len(cmd.arguments) != 2 {
		return fmt.Errorf("Usage: ./app login <username>")
	}

	s.config.CurrentUser = cmd.arguments[1]

	_, err := s.db.GetUser(context.Background(), cmd.arguments[1])
	if err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return fmt.Errorf("Could not find username. Run ./app register <username> first")
		}
		return err
	}

	err = config.SetUser(*s.config)
	if err != nil {
		return fmt.Errorf("Failed to set new user. Error: %v\n", err)
	}
	fmt.Printf("New user: %s\n", s.config.CurrentUser)

	return nil
}

func (c *commands) handleRegister(s *state, cmd command) error {
	if len(cmd.arguments) < 2 {
		return fmt.Errorf("Usage: ./app register <username>\n")
	}
	
	_, err := s.db.CreateUser(context.Background(), database.CreateUserParams {
		ID: uuid.New().String(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name: cmd.arguments[1],
	})
	if err != nil {
		exists := strings.Contains(err.Error(), "duplicate key value violates unique constraint")
		if exists {
			return fmt.Errorf("Username already exists.\n")
		}
		return fmt.Errorf("Failed to register user to DB. Error:\n%v\n", err)
	}
	
	s.config.CurrentUser = cmd.arguments[1]
	err = config.SetUser(*s.config)
	if err != nil {
		return err
	}
	fmt.Printf("New user: %s\n", s.config.CurrentUser)

	return nil
}


func middlewareLoginIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func (s *state, cmd command) error {
		if s.config.CurrentUser == "" {
			return fmt.Errorf("No user logged in. Run ./app register <username> first")
		}

		user, err := s.db.GetUser(context.Background(), s.config.CurrentUser)
		if err != nil {
			if strings.Contains(err.Error(), "no rows in result set") {
				return fmt.Errorf("Could not find username. Run ./app register <username> first")
			}
			return fmt.Errorf("Failed to get current user from database. Error:\n%v\n", err)
		}

		return handler(s, cmd, user)
	}
}


func (c *commands) handleUsers(s *state, _ command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return fmt.Errorf("Could not find username. Run ./app register <username> first")
		}
		return err
	}

	for _, name := range users {
		if name == s.config.CurrentUser {
			fmt.Printf(" * %s (current)\n", name)
		} else {
			fmt.Printf(" * %s\n", name)
		}
	}

	return nil
}

func scrapeFeeds (s *state) error {
	nextFeed, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return fmt.Errorf("Failed to fetch next feed from database. Error: %v", err)
	}
	fmt.Printf("Scraping url: %s ...\n", nextFeed.Url)

	rssFeed, err := parser.FetchFeed(context.Background(), nextFeed.Url)
	if err != nil {
		return fmt.Errorf("Failed to fetch the feed from url: %s. Error: %v", nextFeed.Url, err)
	}

	_, err = s.db.MarkFeedFetched(context.Background(), database.MarkFeedFetchedParams{
		ID: nextFeed.ID,
		LastFetchedAt: sql.NullTime{
			Time: time.Now(),
			Valid: true,
		},
	})	
	if err != nil {
		return err
	}

	fmt.Printf("Channel title: %s\n", rssFeed.Channel.Title)
	for _, f := range rssFeed.Channel.Item {
		pubDate, err := parser.ParseDate(f.PubDate)
		pubNullTime := sql.NullTime{
			Time: pubDate,
			Valid: true,
		}

		if err != nil {
			fmt.Printf("Post %s, has unregonized time format: %s\nError: %v\n", f.Title, f.PubDate, err )
			pubNullTime = sql.NullTime {
				Time: time.Now(),
				Valid: false,
			}
		}
		_, err = s.db.CreatePost(context.Background(), database.CreatePostParams{
			ID: uuid.New().String(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Title: f.Title,
			Url: f.Link,
			Description: sql.NullString{
				String: f.Description,
				Valid: true,
			},
			PublishedAt: pubNullTime,
			FeedID: nextFeed.ID,
		})
		if err != nil {
			if strings.Contains(err.Error(), "unique constraint") {
				continue
			}
			fmt.Printf("Failed to save post %s. Error: %v\n", f.Title, err)
		}
	}

	return nil
}

func (c *commands) handleAgg(s *state, cmd command) error {
	if len(cmd.arguments) != 2 {
		return fmt.Errorf("Usage: ./app agg <time>  --- Ex: 1s, 1m, 1h")
	}

	timeBetweenRequests, err := time.ParseDuration(cmd.arguments[1])
	if err != nil {
		return fmt.Errorf("Time format not accepted. Use one of the following formats: 1s, 1m, 1h\n.Error: %v\n", err)
	}

	fmt.Printf("Collecting feeds every: %v\n", timeBetweenRequests)
	ticker := time.NewTicker(timeBetweenRequests)

	for ; ; <-ticker.C {
		if err := scrapeFeeds(s); err != nil {
			return err
		}
	}
}

func (c *commands) handleReset (s *state, _ command) error {
	err := s.db.Reset(context.Background())
	if err != nil {
		return err
	}

	fmt.Printf("All users and related records were erased!\n")

	return nil
}

func (c *commands) handleAddFeed (s *state, cmd command, user database.User) error {
	if len(cmd.arguments) != 3 {
		return fmt.Errorf("Usage: ./app addfeed <title> <url>")
	}

	feed, err := s.db.CreateFeed(context.Background(), database.CreateFeedParams{
		ID: uuid.New().String(),
		UserID: user.ID,
		Name: cmd.arguments[1],
		Url: cmd.arguments[2],
	})
	if err != nil {
		return fmt.Errorf("Failed to create rss feed in database. Error: %s\n", err)
	}

	feedFollow, err := s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID: uuid.New().String(),
		UserID: user.ID,
		FeedID: feed.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		return fmt.Errorf("Failed follow feed %s. Error: %s\n",feed.Name, err)
	}

	fmt.Printf("Feed %s added for user %s\n", feedFollow.FeedName, s.config.CurrentUser)
	return nil
}

func (c *commands) handleFeeds (s *state, _ command) error {
	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		return fmt.Errorf("Failed to get feeds from database. Error: %v\n", err)
	}

	fmt.Println("Feeds:")
	for i, v := range feeds {
		fmt.Printf("%d. %s -- %s -- [%s]\n",i+1, v.Name, v.Url, v.Username)
	}
	
	return nil
}

func (c *commands) handleFollow (s *state, cmd command, user database.User) error {
	if len(cmd.arguments) != 2 {
		return fmt.Errorf("Usage: ./app follow <url>\n")
	}

	feedId, err := s.db.GetIdFeedByUrl(context.Background(), cmd.arguments[1])
	if err != nil {
		if strings.Contains(err.Error(), "no rows in result") {
			return fmt.Errorf("Feed url not found. Please add it first\n")
		}
		return err
	}

	//insert new feed for current user
	feedFollow, err := s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID: uuid.New().String(),	
		UserID: user.ID,
		FeedID: feedId,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return fmt.Errorf("Current user - %s - already follows feed url: %s\n", user.Name, cmd.arguments[1])
		}
		return err
	}
	//print the name of the feed and the current user once the record is created

	fmt.Printf("User: %s, now follows feed: %s\n", feedFollow.FeedName, feedFollow.UserName)
	
	return nil
}


func (c *commands) handleUnfollow (s *state, cmd command, user database.User) error {
	if len(cmd.arguments) != 2 {
		return fmt.Errorf("Usage: ./app unfollow <url>\n")
	}

	feedId, err := s.db.GetIdFeedByUrl(context.Background(), cmd.arguments[1])
	if err != nil {
		if strings.Contains(err.Error(), "no rows in result") {
			return fmt.Errorf("Feed url not found. Please add it first\n")
		}
		return err
	}

	err = s.db.DeleteFeedFollow(context.Background(), database.DeleteFeedFollowParams{
		UserID: user.ID,
		FeedID: feedId,
	})
	if err != nil {
		return err
	}

	fmt.Printf("User: %s, now unfollows feed: %s\n", user.Name, cmd.arguments[1])
	
	return nil
}

func (c *commands) handleFollowing (s *state, cmd command, user database.User) error {
	feeds, err := s.db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return fmt.Errorf("User %s is not following any feeds\n", user.Name)
		}
		return err
	}

	fmt.Printf("User %s is following %d feeds\n", user.Name, len(feeds))
	for i, f := range feeds {
		fmt.Printf("%d. %s - %s\n", i+1, f.FeedName, f.Url)
	}

	return nil
}

func (c *commands) handleBrowse (s *state, cmd command, user database.User) error {
	var limit int32 = 2
	if len(cmd.arguments) == 2 {
		parsedLimit, err := strconv.ParseInt(cmd.arguments[1], 10, 32)
		if err != nil {
			return fmt.Errorf("Provided limit has the wrong format\n")
		} else {
			limit = int32(parsedLimit)
		}
	}

	posts, err := s.db.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		ID: user.ID,
		Limit: limit,
	})
	if err != nil {
		return fmt.Errorf("Failed to get the posts for the user. Error: %v\n", err)
	}

	fmt.Printf("%+v \n\n", posts)

	for i, p := range posts {
		fmt.Printf("%d. %s - %s\n", i, p.Title, p.Url)
	}

	return nil
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.c[name] = f
}

func (c *commands) run (s *state, cmd command) error {
	currentCmd, ok := c.c[cmd.name]
	if !ok {
		return fmt.Errorf("The command was not found")
	}

	err := currentCmd(s, cmd)
	if err != nil {
		return fmt.Errorf("Failed to run command. Error:\n%w", err)
	}
	return nil
}

func main() {
	activeCfg, err := config.Read()
	if err != nil {
		log.Fatalf("Failed to read the config. Error: %v\n", err)
	}

	db, err := sql.Open("postgres", activeCfg.DB_URL)
	if err != nil {
		log.Fatalf("Failed to open database. Error: %v\n", err)
	}

	dbQuery := database.New(db)
	st := state{
		config: &activeCfg,
		db:		dbQuery,
	}

	activeCommands := commands {
		c: make(map[string]func(*state, command) error),
	}
	activeCommands.register("login", activeCommands.handleLogin)
	activeCommands.register("register", activeCommands.handleRegister)
	activeCommands.register("reset", activeCommands.handleReset)
	activeCommands.register("users", activeCommands.handleUsers)
	activeCommands.register("agg", activeCommands.handleAgg)
	activeCommands.register("addfeed", middlewareLoginIn(activeCommands.handleAddFeed))
	activeCommands.register("feeds", activeCommands.handleFeeds)
	activeCommands.register("follow", middlewareLoginIn(activeCommands.handleFollow))
	activeCommands.register("unfollow", middlewareLoginIn(activeCommands.handleUnfollow))
	activeCommands.register("following", middlewareLoginIn(activeCommands.handleFollowing))
	activeCommands.register("browse", middlewareLoginIn(activeCommands.handleBrowse))

	args := os.Args
	if len(args) < 2 {
		log.Fatalf("The program requires at least one argument\n")
	}
	
	err = activeCommands.run(&st, command{
		name: args[1],
		arguments: args[1:],
	})
	if err != nil {
		log.Fatalf("%v\n", err)
		os.Exit(1)
	}
}
