package bot

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/automuteus/automuteus/v8/internal/server"
	"github.com/automuteus/automuteus/v8/pkg/settings"
	"github.com/bwmarrin/discordgo"
)

// bumped for public rollout. Don't need to update the status message more than once every 2 secs prob
const DeferredEditSeconds = 2
const colorSelectID = "select-color"

type GameStateMessage struct {
	MessageID        string `json:"messageID"`
	MessageChannelID string `json:"messageChannelID"`
	LeaderID         string `json:"leaderID"`
	CreationTimeUnix int64  `json:"creationTimeUnix"`
}

func MakeGameStateMessage() GameStateMessage {
	return GameStateMessage{
		MessageID:        "",
		MessageChannelID: "",
		LeaderID:         "",
		CreationTimeUnix: 0,
	}
}

func (gsm *GameStateMessage) Exists() bool {
	return gsm.MessageID != "" && gsm.MessageChannelID != ""
}

func (dgs *GameState) DeleteGameStateMsg(s *discordgo.Session, reset bool) bool {
	retValue := false
	if dgs.GameStateMsg.Exists() {
		err := s.ChannelMessageDelete(dgs.GameStateMsg.MessageChannelID, dgs.GameStateMsg.MessageID)
		if err != nil {
			retValue = false
		} else {
			retValue = true
		}
	}
	// whether or not we were successful in deleting the message, reset the state
	if reset {
		dgs.GameStateMsg = MakeGameStateMessage()
	}
	return retValue
}

var DeferredEdits = make(map[string]*discordgo.MessageEmbed)
var DeferredEditsLock = sync.Mutex{}

// Note this is not a pointer; we never expect the underlying DGS to change on an edit
func (dgs GameState) dispatchEdit(s *discordgo.Session, me *discordgo.MessageEmbed) (newEdit bool) {
	if !ValidFields(me) {
		return false
	}

	DeferredEditsLock.Lock()

	// if it isn't found, then start the worker to wait to start it (this is a UNIQUE edit)
	if _, ok := DeferredEdits[dgs.GameStateMsg.MessageID]; !ok {
		go deferredEditWorker(s, dgs.GameStateMsg.MessageChannelID, dgs.GameStateMsg.MessageID)
		newEdit = true
	}
	// whether or not it's found, replace the contents with the new message
	DeferredEdits[dgs.GameStateMsg.MessageID] = me
	DeferredEditsLock.Unlock()
	return newEdit
}

func (dgs GameState) shouldRefresh() bool {
	// discord dictates that we can't edit messages that are older than 1 hour
	return (time.Now().Sub(time.Unix(dgs.GameStateMsg.CreationTimeUnix, 0))) > time.Hour
}

func ValidFields(me *discordgo.MessageEmbed) bool {
	for _, v := range me.Fields {
		if v == nil {
			return false
		}
		if v.Name == "" || v.Value == "" {
			return false
		}
	}
	return true
}

func RemovePendingDGSEdit(messageID string) {
	DeferredEditsLock.Lock()
	delete(DeferredEdits, messageID)
	DeferredEditsLock.Unlock()
}

func deferredEditWorker(s *discordgo.Session, channelID, messageID string) {
	time.Sleep(time.Second * time.Duration(DeferredEditSeconds))

	DeferredEditsLock.Lock()
	me := DeferredEdits[messageID]
	delete(DeferredEdits, messageID)
	DeferredEditsLock.Unlock()

	if me != nil {
		editMessageEmbed(s, channelID, messageID, me)
	}
}

// ===== 色ボタンのラベル（絵文字＋カタカナ） =====
//  ['red','🟥 レッド'],   ['black','⬛ ブラック'], ['white','⬜ ホワイト'], ['rose','🌸 ローズ'],
//  ['blue','🔵 ブルー'],  ['cyan','🟦 シアン'],   ['yellow','🟨 イエロー'], ['pink','💗 ピンク'],
//  ['purple','🟣 パープル'], ['orange','🟧 オレンジ'], ['banana','🍌 バナナ'], ['coral','🧱 コーラル'],
//  ['lime','🥬 ライム'],  ['green','🌲 グリーン'], ['gray','⬜ グレー'],   ['maroon','🍷 マルーン'],
//  ['brown','🤎 ブラウン'], ['tan','🟫 タン']

func colorDisplayLabel(value string) string {
	switch strings.ToLower(value) {
	case "red":
		return "🟥 レッド"
	case "black":
		return "⬛ ブラック"
	case "white":
		return "⬜ ホワイト"
	case "rose":
		return "🌸 ローズ"
	case "blue":
		return "🔵 ブルー"
	case "cyan":
		return "🟦 シアン"
	case "yellow":
		return "🟨 イエロー"
	case "pink":
		return "💗 ピンク"
	case "purple":
		return "🟣 パープル"
	case "orange":
		return "🟧 オレンジ"
	case "banana":
		return "🍌 バナナ"
	case "coral":
		return "🧱 コーラル"
	case "lime":
		return "🥬 ライム"
	case "green":
		return "🌲 グリーン"
	case "gray":
		return "⬜ グレー"
	case "maroon":
		return "🍷 マルーン"
	case "brown":
		return "🤎 ブラウン"
	case "tan":
		return "🟫 タン"
	default:
		// 想定外の色はとりあえずそのまま（英語など）
		return value
	}
}

// ===== ボタン式 色選択付きの CreateMessage =====

func (dgs *GameState) CreateMessage(s *discordgo.Session, me *discordgo.MessageEmbed, channelID string, authorID string) bool {
	// 元のセレクトメニュー用関数から色一覧＋絵文字を取得
	opts := EmojisToSelectMenuOptions(GlobalAlivenessEmojis[true], X)

	const maxPerRow = 5
	var components []discordgo.MessageComponent
	curRow := discordgo.ActionsRow{}
	count := 0

	// ✖ 外すボタンを最後に 1 行だけ出したいので、一時退避
	var unlinkBtn *discordgo.Button

	for _, opt := range opts {
		value := opt.Value

		// Unlink 用のオプションは専用ボタンにする
		if value == UnlinkEmojiName {
			btn := discordgo.Button{
				CustomID: fmt.Sprintf("%s:%s", colorSelectID, value),
				Label:    "✖ 外す",
				Style:    discordgo.DangerButton,
				// Emoji は付けない（テキストだけ）
			}
			unlinkBtn = &btn
			continue
		}

		label := colorDisplayLabel(value)

		customID := fmt.Sprintf("%s:%s", colorSelectID, value)

		// ★ ボタン左にキャラ絵文字を出したくないので Emoji は設定しない
		btn := discordgo.Button{
			CustomID: customID,
			Label:    label,
			Style:    discordgo.SecondaryButton,
		}

		curRow.Components = append(curRow.Components, btn)
		count++

		// 5 個ごとに改行
		if count%maxPerRow == 0 {
			components = append(components, curRow)
			curRow = discordgo.ActionsRow{}
		}
	}

	// 余りがあれば最後の行として追加
	if len(curRow.Components) > 0 {
		components = append(components, curRow)
	}

	// ✖ 外すボタンを一番下に 1 行で追加
	if unlinkBtn != nil {
		components = append(components, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				*unlinkBtn,
			},
		})
	}

	msg := sendEmbedWithComponents(s, channelID, me, components)
	if msg != nil {
		dgs.GameStateMsg.LeaderID = authorID
		dgs.GameStateMsg.MessageChannelID = msg.ChannelID
		dgs.GameStateMsg.MessageID = msg.ID
		dgs.GameStateMsg.CreationTimeUnix = time.Now().Unix()
		return true
	}
	return false
}

// ===== ここまで CreateMessage =====

func (bot *Bot) DispatchRefreshOrEdit(readOnlyDgs *GameState, dgsRequest GameStateRequest, sett *settings.GuildSettings) {
	if readOnlyDgs.shouldRefresh() {
		bot.RefreshGameStateMessage(dgsRequest, sett)
	} else {
		edited := readOnlyDgs.dispatchEdit(bot.PrimarySession, bot.gameStateResponse(readOnlyDgs, sett))
		if edited {
			server.RecordDiscordRequests(bot.RedisInterface.client, server.MessageEdit, 1)
		}
	}
}
