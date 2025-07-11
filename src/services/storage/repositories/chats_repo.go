package repositories

import (
	"encoding/json"
	"os"
	"path/filepath"

	"aichat/src/models"
)

const chatsConfigPath = "src/.config/chats.json"

type ChatRepository struct {
	file string
}

func NewChatRepository() *ChatRepository {
	return &ChatRepository{file: chatsConfigPath}
}

func (r *ChatRepository) GetAll() ([]models.ChatFile, error) {
	data, err := os.ReadFile(r.file)
	if err != nil {
		if os.IsNotExist(err) {
			return []models.ChatFile{}, nil
		}
		return nil, err
	}
	var chats []models.ChatFile
	if err := json.Unmarshal(data, &chats); err != nil {
		return nil, err
	}
	return chats, nil
}

func (r *ChatRepository) SaveAll(chats []models.ChatFile) error {
	if err := os.MkdirAll(filepath.Dir(r.file), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(chats, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.file, data, 0600)
}

func (r *ChatRepository) Add(chat models.ChatFile) error {
	chats, err := r.GetAll()
	if err != nil {
		return err
	}
	chats = append(chats, chat)
	return r.SaveAll(chats)
}

func (r *ChatRepository) Remove(title string) error {
	chats, err := r.GetAll()
	if err != nil {
		return err
	}
	newChats := make([]models.ChatFile, 0, len(chats))
	for _, c := range chats {
		if c.Metadata.Title != title {
			newChats = append(newChats, c)
		}
	}
	return r.SaveAll(newChats)
}

func (r *ChatRepository) GetByTitle(title string) (*models.ChatFile, error) {
	chats, err := r.GetAll()
	if err != nil {
		return nil, err
	}
	for _, c := range chats {
		if c.Metadata.Title == title {
			return &c, nil
		}
	}
	return nil, os.ErrNotExist
}
