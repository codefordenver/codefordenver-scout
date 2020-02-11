# Scout Bot
## Commands
General Usage:
```
![command name] [arguments]
```
Commands:
- `!onboardall` - Converts all users with new or onboarding role to member role
- `!onboard` - Converts all users with onboarding role to member role
- `!agenda` - Fetches the agenda for the next meeting, or creates and returns it if it does not exist
- `!join [project name]` - Adds user to the project
- `!leave [project name]` - Removes user from the project
- `!maintain [project name]` - Moves a project to maintenance
- `!track [file name] [link]` - Adds the file to Airtable
- `!untrack [file name]` - Removes the file from Airtable
- `!fetch [file name]` - Fetches a file specified in Airtable

Note: Commands can also be triggered by `@Scout [command]`
## Development
### Setup
Install:
- [Golang](https://golang.org/)
- [go-watcher](https://github.com/canthefason/go-watcher)
- [sops](https://github.com/mozilla/sops)

Set environment variables:
```
AWS_ACCESS_KEY_ID=AWS key ID for sops
AWS_SECRET_ACCESS_KEY=AWS key secret for sops
SCOUT_TOKEN=discord bot token
GDRIVE_CREDS=base64 str version of credentials.json
GDRIVE_ACCESS_TOKEN=base64 str version of token.json
GITHUB_CREDS=base64 str version of github pem file
SCOUT_DB_URL=host for database
SCOUT_DB_PORT=port for database
SCOUT_DB_USER=user for database
SCOUT_DB_NAME=name of database
SCOUT_DB_PASSWORD=password for database
```
Get these values from a current project member

View `models/` for the database schema.

To run the bot, simply navigate to the project directory and run:
```
watcher
```
