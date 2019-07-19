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
- `!fetch [file name]` - Fetches a file specified in the config
## Development
### Setup
Install:
- [Golang](https://golang.org/)
- [go-watcher](https://github.com/canthefason/go-watcher)

Set environment variables:
```
SCOUT_TOKEN=discord bot token
GDRIVE_CREDS=base64 str version of credentials.json
GDRIVE_ACCESS_TOKEN=base64 str version of token.json
GITHUB_CREDS=base64 str version of github pem file
SCOUT_LOCATION_STRING=
```
Get these values from a current project member

To run the bot, simply navigate to the project directory and run:
```
watcher
```
