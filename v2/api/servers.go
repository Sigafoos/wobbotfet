package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/bwmarrin/discordgo"
)

// A Server represents a Discord guild/server.
type Server struct {
	*discordgo.UserGuild
	ParsedPermissions []string
}

// GetServers returns a list of servers the bot is connected to.
func (a *API) GetServers(w http.ResponseWriter, r *http.Request) {
	s, err := a.bot.Servers()
	if err != nil {
		log.Printf("error getting servers: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var servers []*Server
	for _, v := range s {
		servers = append(servers, &Server{
			v,
			parsePermissions(v.Permissions),
		})
	}

	b, err := json.Marshal(servers)
	if err != nil {
		log.Printf("error marshalling json: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(b)
}

func parsePermissions(p int) []string {
	permissions := []string{}
	if p&discordgo.PermissionSendMessages > 0 {
		permissions = append(permissions, "send messages")
	}

	if p&discordgo.PermissionViewChannel > 0 {
		permissions = append(permissions, "view channel")
	}

	if p&discordgo.PermissionSendTTSMessages > 0 {
		permissions = append(permissions, "send tts messages")
	}

	if p&discordgo.PermissionManageMessages > 0 {
		permissions = append(permissions, "manage messages")
	}

	if p&discordgo.PermissionEmbedLinks > 0 {
		permissions = append(permissions, "embed links")
	}

	if p&discordgo.PermissionAttachFiles > 0 {
		permissions = append(permissions, "attach files")
	}

	if p&discordgo.PermissionReadMessageHistory > 0 {
		permissions = append(permissions, "read message history")
	}

	if p&discordgo.PermissionMentionEveryone > 0 {
		permissions = append(permissions, "mention everyone")
	}

	if p&discordgo.PermissionUseExternalEmojis > 0 {
		permissions = append(permissions, "use external emoji")
	}

	if p&discordgo.PermissionVoiceConnect > 0 {
		permissions = append(permissions, "voice connect")
	}

	if p&discordgo.PermissionVoiceSpeak > 0 {
		permissions = append(permissions, "voice speak")
	}

	if p&discordgo.PermissionVoiceMuteMembers > 0 {
		permissions = append(permissions, "voice mute members")
	}

	if p&discordgo.PermissionVoiceDeafenMembers > 0 {
		permissions = append(permissions, "voice deafen members")
	}

	if p&discordgo.PermissionVoiceMoveMembers > 0 {
		permissions = append(permissions, "voice move members")
	}

	if p&discordgo.PermissionVoiceUseVAD > 0 {
		permissions = append(permissions, "voice use vad")
	}

	if p&discordgo.PermissionVoicePrioritySpeaker > 0 {
		permissions = append(permissions, "voice priority speaker")
	}

	if p&discordgo.PermissionChangeNickname > 0 {
		permissions = append(permissions, "change nickname")
	}

	if p&discordgo.PermissionManageNicknames > 0 {
		permissions = append(permissions, "manage nicknames")
	}

	if p&discordgo.PermissionManageRoles > 0 {
		permissions = append(permissions, "manage roles")
	}

	if p&discordgo.PermissionManageWebhooks > 0 {
		permissions = append(permissions, "manage webhooks")
	}

	if p&discordgo.PermissionManageEmojis > 0 {
		permissions = append(permissions, "manage emoji")
	}

	if p&discordgo.PermissionCreateInstantInvite > 0 {
		permissions = append(permissions, "create instant invite")
	}

	if p&discordgo.PermissionKickMembers > 0 {
		permissions = append(permissions, "kick members")
	}

	if p&discordgo.PermissionBanMembers > 0 {
		permissions = append(permissions, "ban members")
	}

	if p&discordgo.PermissionAdministrator > 0 {
		permissions = append(permissions, "administrator")
	}

	if p&discordgo.PermissionManageChannels > 0 {
		permissions = append(permissions, "manage channels")
	}

	if p&discordgo.PermissionManageServer > 0 {
		permissions = append(permissions, "manage server")
	}

	if p&discordgo.PermissionAddReactions > 0 {
		permissions = append(permissions, "add reactions")
	}

	if p&discordgo.PermissionViewAuditLogs > 0 {
		permissions = append(permissions, "view audit logs")
	}

	return permissions
}
