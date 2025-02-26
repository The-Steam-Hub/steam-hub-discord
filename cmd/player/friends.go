package player

import (
	"fmt"
	"math"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/the-steam-hub/discord-bot/cmd"
	"github.com/the-steam-hub/discord-bot/steam"
)

const (
	cap = 50
)

type FriendData struct {
	Friend steam.Friend
	Player steam.Player
}

func PlayerFriends(session *discordgo.Session, interaction *discordgo.InteractionCreate, steamClient steam.Steam, input string) {
	logs := logrus.Fields{
		"input":  input,
		"author": interaction.Member.User.Username,
		"uuid":   uuid.New(),
	}

	id, err := steamClient.ResolveSteamID(input)
	if err != nil {
		logs["error"] = err
		errMsg := "unable to resolve player ID"
		logrus.WithFields(logs).Error(errMsg)
		cmd.HandleMessageError(session, interaction, &logs, errMsg)
		return
	}

	player, err := steamClient.PlayerSummaries(id)
	if err != nil {
		logs["error"] = err
		errMsg := "unable to retrieve player summary"
		logrus.WithFields(logs).Error(errMsg)
		cmd.HandleMessageError(session, interaction, &logs, errMsg)
		return
	}

	friendsList, err := steamClient.FriendsList(player[0].SteamID)
	if err != nil {
		logs["error"] = err
		logrus.WithFields(logs).Error("unable to retrieve friends list")
	}

	// Sorting the friends list so we display the oldest friends first
	sortedFriendsList := steam.FriendsSort(friendsList)
	// Capping the friends list to avoid message overflow issues with Discord
	sortedCappedFriendsList := sortedFriendsList[:int(math.Min(float64(len(sortedFriendsList)), cap))]
	// Length may be zero if the players account is private
	if len(sortedCappedFriendsList) > 0 {
		// Assigning the newest friend to the last index. This allows us to grab the name of the newest friend in the same API call as the other 49 friends
		sortedCappedFriendsList[len(sortedCappedFriendsList)-1] = sortedFriendsList[len(sortedFriendsList)-1]
	}

	// Getting player information for all friends within the cap range
	players, err := steamClient.PlayerSummaries(steam.FriendIDs(sortedFriendsList)[:len(sortedCappedFriendsList)]...)
	if err != nil {
		logs["error"] = err
		logrus.WithFields(logs).Error("unable to retrieve player summary")
	}

	// Friend data and Player data exists in two seperate API calls, and so, we need to tie the data together
	// The data is already sorted and is persisted in the friendData slice
	friendData := make([]FriendData, len(players))
	for _, v := range players {
		for k, j := range sortedCappedFriendsList {
			if v.SteamID == j.ID {
				friendData[k] = FriendData{
					Friend: j,
					Player: v,
				}
			}
		}
	}

	names, statuses, friendsSince, oldest, newest := "", "", "", "", ""

	// Length may be zero if the players account is private
	if len(sortedCappedFriendsList) > 0 {
		oldest = sortedCappedFriendsList[0].ID
		newest = sortedCappedFriendsList[len(sortedCappedFriendsList)-1].ID
	}

	for k, i := range friendData {
		// Avoid adding the last entry (newest friend) if we are at the cap
		if k < cap-1 {
			names += fmt.Sprintf("%s\n", i.Player.Name)
			statuses += fmt.Sprintf("%s\n", i.Player.Status())
			friendsSince += fmt.Sprintf("%s\n", steam.UnixToDate(i.Friend.FriendsSince))
		}
		// Finding the name that belongs to the newest ID
		if i.Player.SteamID == newest {
			newest = i.Player.Name
		}
		// Finding the name that belongs to the oldest ID
		if i.Player.SteamID == oldest {
			oldest = i.Player.Name
		}
	}

	embMsg := &discordgo.MessageEmbed{
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Friend information is dependent upon the user's privacy settings.",
		},
		Color: 0x66c0f4,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: player[0].AvatarFull,
		},
		Author: &discordgo.MessageEmbedAuthor{
			Name: fmt.Sprintf("%s %s", player[0].Status(), player[0].Name),
			URL:  player[0].ProfileURL,
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Newest",
				Value:  cmd.HandleStringDefault(newest),
				Inline: true,
			},
			{
				Name:   "Oldest",
				Value:  cmd.HandleStringDefault(oldest),
				Inline: true,
			},
			{
				Name:   "Count",
				Value:  fmt.Sprintf("%d", len(sortedFriendsList)),
				Inline: true,
			},
			{
				Name:   "Top 50 Friends",
				Value:  cmd.HandleStringDefault(names),
				Inline: true,
			},
			{
				Name:   "Friends For",
				Value:  cmd.HandleStringDefault(friendsSince),
				Inline: true,
			},
			{
				Name:   "Status",
				Value:  cmd.HandleStringDefault(statuses),
				Inline: true,
			},
		},
	}
	cmd.HandleMessageOk(embMsg, session, interaction, &logs)
}
