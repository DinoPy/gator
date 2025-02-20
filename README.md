Requirements:
- Postgres
- Golang

How to install:
- `go install github.com/dinopy/gator`

Local Requirements:
- In ~/ folder you'll need a .gatorconfig.json configured as such:

```
{
    "db_url":"protocol://username:password@host:port/database",
    "current_user_name":"user_name"
}
```

Available commands:
- login - Locates the user in the database & updates the .config file for the current user.
    - Usage: ./app login <name>
- register - Registers the user in the database and runs the login commmand for that user.
    - Usage: ./app register <name>
- reset - Deletes all user related data.
    - Usage: ./app reset
- users - Lists all the users while also labeling the current user.
    - Usage: ./app users
- agg - Runs the scraper function at every 'nmns' time.
    - Usage: ./app agg <time>  --- Ex: 1s, 1m, 1h
- addfeed - Adds the feed to the database & connects it to the user.
    - Usage: ./app addfeed <title> <url>
- feeds - Fetches all the feeds from the database and prints all of them.
    - Usage: ./app feeds
- follow - Takes a link and connect the feed with the user.
    - Usage: ./app follow <url>
- unfollow - Opposite of follow.
    - Usage: ./app unfollow <url>
- following - Lists all fedes that are followed by the current user.
    - Usage: ./app following
- browse - Lists a number of posts that are being followed by the user (through the feeds).
    - Usage: ./app browse <optional_number>

