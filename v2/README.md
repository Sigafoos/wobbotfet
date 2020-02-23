# Wobbotfet!
A Discord bot for reporting the PVP IVs of a Pokemon in Pokemon Go.

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

#### PVP
* `pvp register` to sign up for remote battles on a server: it will PM you to ask for friend code/etc (cross server, only need to do once), and then PM you the codes of everyone you need to add. It also PMs everyone else who's registered to let them know you aren't a rando
* `pvp list` (PM only) to see the info of everyone in all your servers
* `pvp ultra todo` (PM only) to see the list of who you need to be ultra friends with
* `pvp ultra (IGN)` to indicate that you're ultra friends with (IGN). They'll be PMed to confirm, and you can only add people you're registered in servers with (no spamming Kieng or Toshi, sorry). Cross server, so you only need to do it with each person once. 
## Building

### Dependencies
* ranking service: for `rank azumarill 4 1 3`, etc
* want service: for wanting/unwanting Pokemon
* pvp service: for PVP functionality

### Running
You'll need to specify a few environment variables:

* `DISCORD_TOKEN`: the bot token generated in the Discord developer console
* `DISCORD_OWNER` (optional): the ID of who you want to get pings when it goes up/down, ie `193777776543662081`
* `RANK_URL`: the hostname of the ranking service (no trailing slask)
* `WANT_URL`: the hostname of the want service (no trailing slash)
* `WANT_BASICUSER` and `WANT_BASICPASS`: if the want service you have set requires basic auth
