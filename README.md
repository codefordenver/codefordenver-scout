# Scout Bot
## Commands
General Usage:
```
![command name] [arguments] [-b brigade] [-p project]
```
Commands:
- `!onboardall` - Converts all users with new or onboarding role to member role
- `!onboard` - Converts all users with onboarding role to member role
- `!agenda` - Fetches the agenda for the next meeting, or creates and returns it if it does not exist
- `!join [project name]` - Adds user to the project
- `!leave [project name]` - Removes user from the project
- `!maintain [project name]` - Moves a project to maintenance
- `!track [file name] [link]` - Adds the specified file to the brigade
- `!untrack [file name]` - Removes the specified file from the brigade
- `!fetch [file name]` - Fetches the specified file for the brigade
- `!in [start time]` - Starts a session at the specified time, or when the command runs.
- `!out [end time]` - Ends your volunteer session at the specified time, or when the command runs.
- `!time [category]` - Fetches time contributed to the specified brigade, project, or the command sender broken down by the specified category.

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

    At this point, you should be able to run the bot with `air` and receive `error fetching brigade, record not found`. To fix this, either create a mock-up Discord server on which to test and populate the brigades table with `INSERT INTO brigades`, or request a database dump file from a project member to restore from using `psql codefordenver-scout_development postgres < [dumpfile location]`
    
Running

1. To run the bot, simply navigate to the project directory and run:
```
air
```
