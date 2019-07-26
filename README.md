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
- `!list-projects` - Messages the user a list of available projects
- `!join [project-name]` - Adds user to the project
- `!leave [project-name]` - Removes user from the project
## Development
### Setup
Install:
- [Golang](https://golang.org/)
- [go-watcher](https://github.com/canthefason/go-watcher)

Set environment variables:
```
SCOUT_TOKEN=discord bot token
NEW_ROLE=
ONBOARDING_ROLE=
MEMBER_ROLE=
ONBOARDING_INVITE_CODE=
CODE_OF_CONDUCT_MESSAGE_ID=
AGENDA_FOLDER_ID=
SCOUT_LOCATION_STRING=
GDRIVE_CREDS=base64 str version of credentials.json
GDRIVE_ACCESS_TOKEN=base64 str version of token.json
GITHUB_CREDS=base64 str version of github pem file
DISCORD_GUILD_ID
PROJECT_CATEGORY_ID
SCOUT_FILES=Mappings of file name and file IDs for fetch
SCOUT_ISSUE_EMOJI
SCOUT_ORG_NAME
```
Get these values from a current project member

To run the bot, simply navigate to the project directory and run:
```
watcher
```
