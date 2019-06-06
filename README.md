# Wobbotfet!
A Discord bot for reporting the PVP IVs of a Pokemon in Pokemon Go.

## Usage
Run `!rank help` in a channel where wobbotfet is present.

To add wobbotfet: https://discordapp.com/oauth2/authorize?client_id=584875596484444175&scope=bot&permissions=2048

## Building

### Dependencies
This contacts a service for its data. Currently it's hard coded to use the Heroku deployment of [Sigafoos/iv](https://github.com/Sigafoos/iv).

### Running
You'll need a `config.yml` (or json, or toml, or whatever [spf13/viper](github.com/spf13/viper) accepts. Using yaml:

```yaml
token:
  prod: abc413
```

Where `abc413` is the bot token generated in the Discord developer console. `prod` is required, but you can have any number of them to run as different bots (for instance, a `dev` bot to test on)
