package bot

import (
	"fmt"
	"os"
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
			Other: "設定一覧",
		}),
		Description: sett.LocalizeMessage(&i18n.Message{
			ID:    "responses.settingResponse.Description",
			Other: "`/settings <項目>` で、以下の設定を変更できます。",
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
			Other: "AutoMuteUs Premium のご利用ありがとうございます！",
		})
	} else {
		desc = sett.LocalizeMessage(&i18n.Message{
			ID:    "responses.settingResponse.PremiumNoThanks",
			Other: "以下は AutoMuteUs Premium 専用の設定です。詳細は `/premium` を実行してください。",
		})
	}
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "\u200B",
		Value:  "\u200B",
		Inline: false,
	})
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "💎 Premium 設定 💎",
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

// ▼ 将来「色名をカタカナ」に変えるとき用のマップ（まだどこからも呼んでいないので安全）
//    ToEmojiEmbedFields などで色名を表示している箇所を触るとき、
//    colorName をそのまま使う代わりに toJPColorName(colorName) を噛ませてください。
var jpColorNames = map[string]string{
	"Red":    "レッド",
	"Blue":   "ブルー",
	"Green":  "グリーン",
	"Pink":   "ピンク",
	"Orange": "オレンジ",
	"Yellow": "イエロー",
	"Black":  "ブラック",
	"White":  "ホワイト",
	"Purple": "パープル",
	"Brown":  "ブラウン",
	"Cyan":   "シアン",
	"Lime":   "ライム",
}

func toJPColorName(en string) string {
	if jp, ok := jpColorNames[en]; ok {
		return jp
	}
	return en
}

func (bot *Bot) gameStateResponse(dgs *GameState, sett *settings.GuildSettings) *discordgo.MessageEmbed {
	// ゲームのフェーズごとに表示を切り替え
	messages := map[game.Phase]func(dgs *GameState, emojis AlivenessEmojis, sett *settings.GuildSettings) *discordgo.MessageEmbed{
		game.MENU:     menuMessage,
		game.LOBBY:    lobbyMessage,
		game.TASKS:    gamePlayMessage,
		game.DISCUSS:  gamePlayMessage,
		game.GAMEOVER: gamePlayMessage,
	}
	return messages[dgs.GameData.Phase](dgs, bot.StatusEmojis, sett)
}

// ──────────────────────────────
// ステータス上部のメタ情報（ホスト/ボイチャ/リンク数）
// ──────────────────────────────

// room, region はもう使わないので _ で捨てる
func lobbyMetaEmbedFields(_ /*room*/, _ /*region*/ string, author, voiceChannelID string, playerCount int, linkedPlayers int, sett *settings.GuildSettings) []*discordgo.MessageEmbedField {
	gameInfoFields := make([]*discordgo.MessageEmbedField, 0)

	// ホスト
	if author != "" {
		gameInfoFields = append(gameInfoFields, &discordgo.MessageEmbedField{
			Name: sett.LocalizeMessage(&i18n.Message{
				ID:    "responses.lobbyMetaEmbedFields.Host",
				Other: "ホスト",
			}),
			Value:  discord.MentionByUserID(author),
			Inline: false,
		})
	}

	// リンク済みメンバー（ホストの直下／改行して表示）
	if linkedPlayers > playerCount {
		linkedPlayers = playerCount
	}
	gameInfoFields = append(gameInfoFields, &discordgo.MessageEmbedField{
		Name: sett.LocalizeMessage(&i18n.Message{
			ID:    "responses.lobbyMetaEmbedFields.PlayersLinked",
			Other: "リンク済みメンバー",
		}),
		Value:  fmt.Sprintf("%v/%v", linkedPlayers, playerCount),
		Inline: false,
	})

	// ボイスチャンネル
	if voiceChannelID != "" {
		gameInfoFields = append(gameInfoFields, &discordgo.MessageEmbedField{
			Name: sett.LocalizeMessage(&i18n.Message{
				ID:    "responses.lobbyMetaEmbedFields.VoiceChannel",
				Other: "ボイスチャンネル",
			}),
			Value:  discord.MentionByChannelID(voiceChannelID),
			Inline: false,
		})
	}

	// ROOM CODE / REGION は一切追加しない（完全に非表示）

	return gameInfoFields
}

// ──────────────────────────────
// メニュー画面（MENU フェーズ）
// ──────────────────────────────

func menuMessage(dgs *GameState, _ AlivenessEmojis, sett *settings.GuildSettings) *discordgo.MessageEmbed {
	var footer *discordgo.MessageEmbedFooter
	desc, color := dgs.descriptionAndColor(sett)
	if color == discord.DEFAULT {
		color = discord.GREEN
		footer = &discordgo.MessageEmbedFooter{
			Text: sett.LocalizeMessage(&i18n.Message{
				ID:    "responses.menuMessage.Linked.FooterText",
				Other: "Among Us でロビーに入ると、試合が自動的に開始されます。",
			}),
			IconURL:      "",
			ProxyIconURL: "",
		}
	}

	fields := make([]*discordgo.MessageEmbedField, 0)
	author := dgs.GameStateMsg.LeaderID
	if author != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name: sett.LocalizeMessage(&i18n.Message{
				ID:    "responses.lobbyMetaEmbedFields.Host",
				Other: "ホスト",
			}),
			Value:  discord.MentionByUserID(author),
			Inline: false,
		})
	}
	if dgs.VoiceChannel != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name: sett.LocalizeMessage(&i18n.Message{
				ID:    "responses.lobbyMetaEmbedFields.VoiceChannel",
				Other: "ボイスチャンネル",
			}),
			Value:  "<#" + dgs.VoiceChannel + ">",
			Inline: false,
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
		Thumbnail:   nil,
		Video:       nil,
		Provider:    nil,
		Author:      nil,
		Fields:      fields,
	}
	return &msg
}

// ──────────────────────────────
// ロビー画面（LOBBY フェーズ）
// ──────────────────────────────

func lobbyMessage(dgs *GameState, emojis AlivenessEmojis, sett *settings.GuildSettings) *discordgo.MessageEmbed {
	room, region, playMap := dgs.GameData.GetRoomRegionMap() // room/region は現在表示に使っていない
	_ = room
	_ = region

	gameInfoFields := lobbyMetaEmbedFields(room, region, dgs.GameStateMsg.LeaderID, dgs.VoiceChannel, dgs.GameData.GetNumDetectedPlayers(), dgs.GetCountLinked(), sett)

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
				Other: "下のボタンから自分の色を選んでください。（× で解除）",
			},
				map[string]interface{}{
					"X": X,
				}),
			IconURL:      "",
			ProxyIconURL: "",
		},
		Color:     color,
		Image:     nil,
		Thumbnail: nil, // 地図は表示しない
		Video:     nil,
		Provider:  nil,
		Author:    nil,
		Fields:    listResp,
	}
	_ = playMap // 使わないが、将来の拡張用に残しておく
	return &msg
}

// ──────────────────────────────
// ゲーム終了メッセージ（GAMEOVER 時のサマリ）
// ──────────────────────────────

func gameOverMessage(dgs *GameState, emojis AlivenessEmojis, sett *settings.GuildSettings, winners string) *discordgo.MessageEmbed {
	_, _, playMap := dgs.GameData.GetRoomRegionMap()

	listResp := dgs.ToEmojiEmbedFields(emojis, sett)

	desc := sett.LocalizeMessage(&i18n.Message{
		ID:    "eventHandler.gameOver.matchID",
		Other: "ゲーム終了！ この試合の Match ID: `{{.MatchID}}`\n{{.Winners}}",
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
				Other: "{{.Mins}} 分後にこのサマリーは自動削除されます。",
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
		Title:       sett.LocalizeMessage(amongus.ToLocale(game.GAMEOVER)),
		Description: desc,
		Timestamp:   time.Now().Format(ISO8601),
		Footer:      footer,
		Color:       discord.DARK_GOLD, // DARK GOLD
		Image:       nil,
		Thumbnail:   nil, // 地図はここでも非表示
		Video:       nil,
		Provider:    nil,
		Author:      nil,
		Fields:      listResp,
	}
	return &msg
}

func getThumbnailFromMap(playMap game.PlayMap, sett *settings.GuildSettings) *discordgo.MessageEmbedThumbnail {
	// いまは使用していないが、将来「地図を戻したい」とき用に関数だけ残しておく
	url := game.FormMapUrl(os.Getenv("BASE_MAP_URL"), playMap, sett.MapVersion == "detailed")
	if url != "" {
		return &discordgo.MessageEmbedThumbnail{
			URL: url,
		}
	}
	return nil
}

// ──────────────────────────────
// ゲーム中（TASKS / DISCUSS / GAMEOVER 中のステータス）
// ──────────────────────────────

func gamePlayMessage(dgs *GameState, emojis AlivenessEmojis, sett *settings.GuildSettings) *discordgo.MessageEmbed {
	phase := dgs.GameData.GetPhase()
	playMap := dgs.GameData.GetPlayMap()

	listResp := dgs.ToEmojiEmbedFields(emojis, sett)
	gameInfoFields := lobbyMetaEmbedFields("", "", dgs.GameStateMsg.LeaderID, dgs.VoiceChannel, dgs.GameData.GetNumDetectedPlayers(), dgs.GetCountLinked(), sett)
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

	// フェーズ名を日本語寄りに
	var title string
	switch phase {
	case game.TASKS:
		title = "タスク中"
	case game.DISCUSS:
		title = "会議中"
	case game.GAMEOVER:
		title = "ゲーム終了"
	default:
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
		Thumbnail:   nil, // 地図は非表示
		Video:       nil,
		Provider:    nil,
		Author:      nil,
		Fields:      listResp,
	}

	_ = playMap // 今は使っていない

	return &msg
}

// returns the description and color to use, based on the gamestate
// usage dictates DEFAULT should be overwritten by other state subsequently,
// whereas RED and DARK_ORANGE are error/flag values that should be passed on
func (dgs *GameState) descriptionAndColor(sett *settings.GuildSettings) (string, int) {
	if !dgs.Linked {
		return sett.LocalizeMessage(&i18n.Message{
			ID:    "responses.notLinked.Description",
			Other: "❌ **キャプチャがリンクされていません！ 上のリンクから接続してください。**",
		}), discord.RED // red
	} else if !dgs.Running {
		return sett.LocalizeMessage(&i18n.Message{
			ID:    "responses.makeDescription.GameNotRunning",
			Other: "⚠ **Bot は一時停止中です。** `/pause` で再開できます。",
		}), discord.DARK_ORANGE
	}
	return "\n", discord.DEFAULT
}

func nonPremiumSettingResponse(sett *settings.GuildSettings) string {
	return sett.LocalizeMessage(&i18n.Message{
		ID:    "responses.nonPremiumSetting.Desc",
		Other: "申し訳ありませんが、その設定は AutoMuteUs Premium 専用です。`/premium` で詳細を確認できます。",
	})
}
