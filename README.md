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
- `!list-projects` - Messages the user a list of available projects
- `!join [project name]` - Adds user to the project
- `!leave [project name]` - Removes user from the project
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
AIRTABLE_API_KEY=Airtable account key
GDRIVE_CREDS=base64 str version of credentials.json
GDRIVE_ACCESS_TOKEN=base64 str version of token.json
GITHUB_CREDS=base64 str version of github pem file
SCOUT_LOCATION_STRING=TZ data location string
```
Get these values from a current project member

Example config.yaml:
```yaml
Brigades:
-   GuildID: "5356701682701XXXXX"
    ActiveProjectCategoryID: "5356777664573XXXXX"
    EveryoneRole: "5356701682701XXXXX" #Yes, this is the same as the GuildID, they are separated for clarity
    NewRole: "5738722616191XXXXX"
    OnboardingRole: "5783212265866XXXXX"
    MemberRole: "5726261632482XXXXX"
    OnboardingInviteCode: "XXXXXX"
    CodeOfConductMessageID: "5802051351128XXXXX"
    AgendaFolderID: "1NL1M9G0iJVwNDa7kRL1rqypUnafXXXXX"
    LocationString: "America/Denver"
    AirtableBaseID: "appXXXXXXXXXXXXXX"
    GithubOrg: "codefordenver"
    IssueEmoji: "➡"

The configuration file should be encrypted via sops. Contact a Code for Denver member to have your configuration info added & encrypted. 

```

To run the bot, simply navigate to the project directory and run:
```
watcher
```
