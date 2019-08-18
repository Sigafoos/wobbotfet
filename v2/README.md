# Wobbotfet!
A Discord bot for reporting the PVP IVs of a Pokemon in Pokemon Go.

**Note**! This documentation is for the original v1 bot. For v2, which adds more commands and has a different syntax, see the v2 directory's README.

## Usage
To add wobbotfet: https://discordapp.com/oauth2/authorize?client_id=584875596484444175&scope=bot&permissions=2048

Interacting with wobbotfet is done by mentioning it. For help: `@wobbotfet help`

### Features
#### IVs
* `rank wobbotfet 12 13 10` for the rank of the IV spread `12/13/10`
* `vrank wobbotfet 12 13 10` for the rank, and also the numbers used to calculate it
* `betterthan wobbotfet 12 13 10` for the odds of a better rank (in a variety of circumstances)

#### Wants
* `want shieldon` to add Shieldon to your want list
* `unwant shieldon` to remove Shieldon
* `wants` to list your wants

If wobbotfet has been granted the ability to manage roles it will add/remove a role with the name of the Pokemon. This will allow you to mention `@shieldon` and notify everyone who wants Shieldon.

The wants are cross-server. However, currently there's no reconciliation of data if you want in one server to add the role in another. A "fix" is to unwant and then want the Pokemon in the server without the role (or vice versa).

## Building

### Dependencies
This contacts services for its data. By default it uses the production deployments of [Sigafoos/iv](https://github.com/Sigafoos/iv) for the rank commands and [Sigafoos/pokewants](https://github.com/sigafoos/pokewants) for want.

These can be overwritten by setting the `wanturl` and `rankurl` environment variables.

### Running
You'll need a `config.yml` (or json, or toml, or whatever [spf13/viper](github.com/spf13/viper) accepts. Using yaml:

```yaml
token:
  prod: abc413
```

Where `abc413` is the bot token generated in the Discord developer console. `prod` is required, but you can have any number of them to run as different bots (for instance, a `dev` bot to test on)
