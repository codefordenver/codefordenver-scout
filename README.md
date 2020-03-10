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
- [PostgreSQL](https://www.postgresql.org/download/)
- [air](https://github.com/cosmtrek/air)

Set environment variables:
```
SCOUT_TOKEN=discord bot token
GDRIVE_CREDS=base64 str version of credentials.json
GDRIVE_ACCESS_TOKEN=base64 str version of token.json
GITHUB_CREDS=base64 str version of github pem file
DATABASE_URL=full postgres connection string
```
Get these values from a current project member

View `models/` for the database schema.

PostgreSQL setup:

1. Create postgres user (if not handled by installation):

   `createuser postgres --superuser --pwprompt`

    And set the password to `postgres` for development.

2. Create Database
   
   `createdb codefordenver-scout_development --owner=postgres`

    And 

Running

1. To run the bot, simply navigate to the project directory and run:
```
air
```
