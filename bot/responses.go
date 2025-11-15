package bot

import (
	"fmt"
	"time"

	"github.com/automuteus/automuteus/v8/bot/setting"
	"github.com/automuteus/automuteus/v8/pkg/amongus"
	"github.com/automuteus/automuteus/v8/pkg/discord"
	"github.com/automuteus/automuteus/v8/pkg/game"
	"github.com/automuteus/automuteus/v8/pkg/settings"
	"github.com/bwmarrin/discordgo"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

const ISO8601 = "2006-01-02T15:04:05-0700"

func settingResponse(settingsList []setting.Setting, sett *settings.GuildSettings, prem bool) *discordgo.MessageEmbed {
	embed := discordgo.MessageEmbed{
		URL:  "",
		Type: "",
		Title: sett.LocalizeMessage(&i18n.Message{
			ID:    "responses.settingResponse.Title",
			Other: "Settings",
		}),
		Description: sett.LocalizeMessage(&i18n.Message{
			ID:    "responses.settingResponse.Description",
			Other: "Type `/settings <setting>` to change a setting from those listed below",
		}),
		Timestamp: "",
		Color:     15844367, // GOLD
		Image:     nil,
		Thumbnail: nil,
		Video:     nil,
		Provider:  nil,
		Author:    nil,
	}

	fields := make([]*discordgo.MessageEmbedField, 0)
	for _, v := range settingsList {
		if !v.Premium {
			name := v.Name
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   name,
				Value:  sett.LocalizeMessage(&i18n.Message{Other: v.ShortDesc}),
				Inline: true,
			})
		}
	}
	var desc string
	if prem {
		desc = sett.LocalizeMessage(&i18n.Message{
			ID:    "responses.settingResponse.PremiumThanks",
			Other: "Thanks for being an AutoMuteUs Premium user!",
		})
	} else {
		desc = sett.LocalizeMessage(&i18n.Message{
			ID:    "responses.settingResponse.PremiumNoThanks",
			Other: "The following settings are only for AutoMuteUs premium users.\nType `/premium` to learn more!",
		})
	}
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "\u200B",
		Value:  "\u200B",
		Inline: false,
	})
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "💎 Premium Settings 💎",
		Value:  desc,
		Inline: false,
	})
	for _, v := range settingsList {
		if v.Premium {
			name := v.Name
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   name,
				Value:  sett.LocalizeMessage(&i18n.Message{Other: v.ShortDesc}),
				Inline: true,
			})
		}
	}

	embed.Fields = fields
	return &embed
}

// ===== ステータス埋め込み本体 =====

func (bot *Bot) gameStateResponse(dgs *GameState, sett *settings.GuildSettings) *discordgo.MessageEmbed {
	// ゲームの状態に応じてメッセージを切り替え
	messages := map[game.Phase]func(dgs *GameState, emojis AlivenessEmojis, sett *settings.GuildSettings) *discordgo.MessageEmbed{
		game.MENU:     menuMessage,
		game.LOBBY:    lobbyMessage,
		game.TASKS:    gamePlayMessage,
		game.DISCUSS:  gamePlayMessage,
		game.GAMEOVER: gamePlayMessage,
	}
	return messages[dgs.GameData.Phase](dgs, bot.StatusEmojis, sett)
}

// --- 上部のメタ情報（ホスト・VC・リンク人数） ---
// room / region はもう使わないので _ で捨てています。
// ラベルは日本語固定にしています。
func lobbyMetaEmbedFields(_ string, _ string, author, voiceChannelID string, playerCount int, linkedPlayers int, _ *settings.GuildSettings) []*discordgo.MessageEmbedField {
	gameInfoFields := make([]*discordgo.MessageEmbedField, 0)

	if author != "" {
		gameInfoFields = append(gameInfoFields, &discordgo.MessageEmbedField{
			Name:   "ホスト",
			Value:  discord.MentionByUserID(author),
			Inline: true,
		})
	}
	if voiceChannelID != "" {
		gameInfoFields = append(gameInfoFields, &discordgo.MessageEmbedField{
			Name:   "ボイスチャンネル",
			Value:  discord.MentionByChannelID(voiceChannelID),
			Inline: true,
		})
	}
	if linkedPlayers > playerCount {
		linkedPlayers = playerCount
	}
	gameInfoFields = append(gameInfoFields, &discordgo.MessageEmbedField{
		Name:   "リンク済みメンバー",
		Value:  fmt.Sprintf("%v/%v", linkedPlayers, playerCount),
		Inline: true,
	})

	return gameInfoFields
}

// ===== メニュー画面 =====

func menuMessage(dgs *GameState, _ AlivenessEmojis, sett *settings.GuildSettings) *discordgo.MessageEmbed {
	var footer *discordgo.MessageEmbedFooter
	desc, color := dgs.descriptionAndColor(sett)
	if color == discord.DEFAULT {
		color = discord.GREEN
		footer = &discordgo.MessageEmbedFooter{
			Text: sett.LocalizeMessage(&i18n.Message{
				ID:    "responses.menuMessage.Linked.FooterText",
				Other: "Among Us でロビーに入ると試合が開始されます。",
			}),
			IconURL:      "",
			ProxyIconURL: "",
		}
	}

	fields := make([]*discordgo.MessageEmbedField, 0)
	author := dgs.GameStateMsg.LeaderID
	if author != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name: "ホスト",
			Value: discord.MentionByUserID(author),
			Inline: true,
		})
	}
	if dgs.VoiceChannel != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "ボイスチャンネル",
			Value:  "<#" + dgs.VoiceChannel + ">",
			Inline: true,
		})
	}
	if len(fields) == 2 {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "\u200B",
			Value:  "\u200B",
			Inline: true,
		})
	}

	msg := discordgo.MessageEmbed{
		URL:  "",
		Type: "",
		Title: sett.LocalizeMessage(&i18n.Message{
			ID:    "responses.menuMessage.Title",
			Other: "メインメニュー",
		}),
		Description: desc,
		Timestamp:   time.Now().Format(ISO8601),
		Footer:      footer,
		Color:       color,
		Image:       nil,
		Thumbnail:   nil, // 地図は表示しない
		Video:       nil,
		Provider:    nil,
		Author:      nil,
		Fields:      fields,
	}
	return &msg
}

// ===== ロビー画面 =====

func lobbyMessage(dgs *GameState, emojis AlivenessEmojis, sett *settings.GuildSettings) *discordgo.MessageEmbed {
	// room, region だけ取得してラベル部分に渡す（中身は使わない）
	room, region, _ := dgs.GameData.GetRoomRegionMap()
	gameInfoFields := lobbyMetaEmbedFields(
		room,
		region,
		dgs.GameStateMsg.LeaderID,
		dgs.VoiceChannel,
		dgs.GameData.GetNumDetectedPlayers(),
		dgs.GetCountLinked(),
		sett,
	)

	listResp := dgs.ToEmojiEmbedFields(emojis, sett)
	listResp = append(gameInfoFields, listResp...)

	desc, color := dgs.descriptionAndColor(sett)
	if color == discord.DEFAULT {
		color = discord.GREEN
	}

	msg := discordgo.MessageEmbed{
		URL:  "",
		Type: "",
		Title: sett.LocalizeMessage(&i18n.Message{
			ID:    "responses.lobbyMessage.Title",
			Other: "ロビー",
		}),
		Description: desc,
		Timestamp:   time.Now().Format(ISO8601),
		Footer: &discordgo.MessageEmbedFooter{
			Text: sett.LocalizeMessage(&i18n.Message{
				ID: "responses.lobbyMessage.Footer.Text",
				Other: "下の色ボタンからアモアスの色を選んでください。（× で解除）",
			}),
			IconURL:      "",
			ProxyIconURL: "",
		},
		Color:     color,
		Image:     nil,
		Thumbnail: nil, // ★ 地図サムネイル削除
		Video:     nil,
		Provider:  nil,
		Author:    nil,
		Fields:    listResp,
	}
	return &msg
}

// ===== ゲーム終了画面 =====

func gameOverMessage(dgs *GameState, emojis AlivenessEmojis, sett *settings.GuildSettings, winners string) *discordgo.MessageEmbed {
	_, _, _ = dgs.GameData.GetRoomRegionMap()

	listResp := dgs.ToEmojiEmbedFields(emojis, sett)

	desc := sett.LocalizeMessage(&i18n.Message{
		ID:    "eventHandler.gameOver.matchID",
		Other: "ゲーム終了！ この試合の統計情報は Match ID: `{{.MatchID}}` から確認できます。\n{{.Winners}}",
	},
		map[string]interface{}{
			"MatchID": matchIDCode(dgs.ConnectCode, dgs.MatchID),
			"Winners": winners,
		})

	var footer *discordgo.MessageEmbedFooter

	if sett.DeleteGameSummaryMinutes > 0 {
		footer = &discordgo.MessageEmbedFooter{
			Text: sett.LocalizeMessage(&i18n.Message{
				ID:    "eventHandler.gameOver.deleteMessageFooter",
				Other: "このメッセージは {{.Mins}} 分後に削除されます。",
			},
				map[string]interface{}{
					"Mins": sett.DeleteGameSummaryMinutes,
				}),
			IconURL:      "",
			ProxyIconURL: "",
		}
	}

	msg := discordgo.MessageEmbed{
		URL:         "",
		Type:        "",
		Title:       "ゲーム終了",
		Description: desc,
		Timestamp:   time.Now().Format(ISO8601),
		Footer:      footer,
		Color:       discord.DARK_GOLD,
		Image:       nil,
		Thumbnail:   nil, // ★ 地図サムネイル削除
		Video:       nil,
		Provider:    nil,
		Author:      nil,
		Fields:      listResp,
	}
	return &msg
}

// ===== プレイ中（タスク / 会議） =====

func gamePlayMessage(dgs *GameState, emojis AlivenessEmojis, sett *settings.GuildSettings) *discordgo.MessageEmbed {
	phase := dgs.GameData.GetPhase()
	_ = dgs.GameData.GetPlayMap()

	listResp := dgs.ToEmojiEmbedFields(emojis, sett)

	gameInfoFields := lobbyMetaEmbedFields(
		"", "",
		dgs.GameStateMsg.LeaderID,
		dgs.VoiceChannel,
		dgs.GameData.GetNumDetectedPlayers(),
		dgs.GetCountLinked(),
		sett,
	)
	listResp = append(gameInfoFields, listResp...)

	desc, color := dgs.descriptionAndColor(sett)
	if color == discord.DEFAULT {
		switch phase {
		case game.TASKS:
			color = discord.BLUE
		case game.DISCUSS:
			color = discord.PURPLE
		}
	}

	// タイトルを日本語にする
	var title string
	switch phase {
	case game.TASKS:
		title = "タスク中"
	case game.DISCUSS:
		title = "会議中"
	case game.GAMEOVER:
		title = "ゲーム終了"
	default:
		// 念のため、既存のローカライズもフォールバックで残しておく
		title = sett.LocalizeMessage(amongus.ToLocale(phase))
	}

	msg := discordgo.MessageEmbed{
		URL:         "",
		Type:        "",
		Title:       title,
		Description: desc,
		Timestamp:   time.Now().Format(ISO8601),
		Color:       color,
		Footer:      nil,
		Image:       nil,
		Thumbnail:   nil, // ★ 地図サムネイル削除
		Video:       nil,
		Provider:    nil,
		Author:      nil,
		Fields:      listResp,
	}

	return &msg
}

// ===== 汎用メッセージ =====

// returns the description and color to use, based on the gamestate
// usage dictates DEFAULT should be overwritten by other state subsequently,
// whereas RED and DARK_ORANGE are error/flag values that should be passed on
func (dgs *GameState) descriptionAndColor(sett *settings.GuildSettings) (string, int) {
	if !dgs.Linked {
		return sett.LocalizeMessage(&i18n.Message{
			ID:    "responses.notLinked.Description",
			Other: "❌ **キャプチャがリンクされていません！ 上のリンクから接続してください。** ❌",
		}), discord.RED
	} else if !dgs.Running {
		return sett.LocalizeMessage(&i18n.Message{
			ID:    "responses.makeDescription.GameNotRunning",
			Other: "\n⚠ **Bot は一時停止中です** ⚠\n\n",
		}), discord.DARK_ORANGE
	}
	return "\n", discord.DEFAULT
}

func nonPremiumSettingResponse(sett *settings.GuildSettings) string {
	return sett.LocalizeMessage(&i18n.Message{
		ID:    "responses.nonPremiumSetting.Desc",
		Other: "申し訳ありませんが、その設定は AutoMuteUs Premium 専用です。詳細は `/premium` を確認してください。",
	})
}
