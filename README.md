#Scout Bot

##Commands
General Usage:
```
![command name] [arguments]
```
Commands:
- `!onboardall` - Converts all users with "new" role to "member" role
##Development

###Setup
Install:
- [Golang](https://golang.org/)
- [go-watcher](https://github.com/canthefason/go-watcher)

Set environment variables:
```
SCOUT_TOKEN
NEW_ROLE
ONBOARDING_ROLE
MEMBER_ROLE
```
Get these values from a current project member

To run the bot, simply navigate to the project directory and run:
```
watcher
```
