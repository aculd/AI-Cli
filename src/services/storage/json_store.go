// Package storage provides JSON-backed repository implementations.
package storage

import (
	"aichat/src/types"
	"encoding/json"
	"os"
	"path/filepath"
)

// --- Chat Repository ---
// Stores each chat as a separate JSON file in .util/chats/
type JSONChatRepository struct {
	dir string // directory for chat files (e.g., .util/chats/)
}

func NewJSONChatRepository(dir string) *JSONChatRepository {
	if dir == "" {
		dir = "src/.config/chats/"
	}
	return &JSONChatRepository{dir: dir}
}

func (r *JSONChatRepository) GetAll() ([]*types.ChatFile, error) {
	files, err := os.ReadDir(r.dir)
	if err != nil {
		return nil, err
	}
	var chats []*types.ChatFile
	for _, f := range files {
		if f.IsDir() || filepath.Ext(f.Name()) != ".json" {
			continue
		}
		chat, err := r.GetByID(f.Name()[:len(f.Name())-5])
		if err == nil && chat != nil {
			chats = append(chats, chat)
		}
	}
	return chats, nil
}

func (r *JSONChatRepository) GetByID(name string) (*types.ChatFile, error) {
	path := filepath.Join(r.dir, name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var chat types.ChatFile
	if err := json.Unmarshal(data, &chat); err != nil {
		return nil, err
	}
	return &chat, nil
}

func (r *JSONChatRepository) Save(chat *types.ChatFile) error {
	if chat == nil || chat.Metadata.Title == "" {
		return os.ErrInvalid
	}
	path := filepath.Join(r.dir, chat.Metadata.Title+".json")
	data, err := json.MarshalIndent(chat, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (r *JSONChatRepository) Delete(name string) error {
	path := filepath.Join(r.dir, name+".json")
	return os.Remove(path)
}

// GetChatFileInfo returns the os.FileInfo for a chat file by name.
func (r *JSONChatRepository) GetChatFileInfo(name string) (os.FileInfo, error) {
	path := filepath.Join(r.dir, name+".json")
	return os.Stat(path)
}

// --- Prompt Repository ---
// Stores all prompts in a single .util/prompts.json file as a JSON array
// Uses types.Prompt

type JSONPromptRepository struct {
	file string // path to prompts.json (e.g., .util/prompts.json)
}

func NewJSONPromptRepository(file string) *JSONPromptRepository {
	if file == "" {
		file = "src/.config/prompts.json"
	}
	return &JSONPromptRepository{file: file}
}

func (r *JSONPromptRepository) GetAll() ([]*types.Prompt, error) {
	data, err := os.ReadFile(r.file)
	if err != nil {
		return nil, err
	}
	var prompts []*types.Prompt
	if err := json.Unmarshal(data, &prompts); err != nil {
		return nil, err
	}
	return prompts, nil
}

func (r *JSONPromptRepository) GetByID(name string) (*types.Prompt, error) {
	prompts, err := r.GetAll()
	if err != nil {
		return nil, err
	}
	for _, p := range prompts {
		if p.Name == name {
			return p, nil
		}
	}
	return nil, os.ErrNotExist
}

func (r *JSONPromptRepository) Save(prompt *types.Prompt) error {
	if prompt == nil || prompt.Name == "" {
		return os.ErrInvalid
	}
	prompts, _ := r.GetAll()
	updated := false
	for i, p := range prompts {
		if p.Name == prompt.Name {
			prompts[i] = prompt
			updated = true
			break
		}
	}
	if !updated {
		prompts = append(prompts, prompt)
	}
	data, err := json.MarshalIndent(prompts, "", "  ")
	if err != nil {
		return err
	}
	tmp := r.file + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, r.file)
}

func (r *JSONPromptRepository) Delete(name string) error {
	prompts, err := r.GetAll()
	if err != nil {
		return err
	}
	var newPrompts []*types.Prompt
	for _, p := range prompts {
		if p.Name != name {
			newPrompts = append(newPrompts, p)
		}
	}
	data, err := json.MarshalIndent(newPrompts, "", "  ")
	if err != nil {
		return err
	}
	tmp := r.file + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, r.file)
}

// --- Model Repository ---
// Stores all models in a single .util/models.json file as a {"models": [...]} object
// Uses types.ModelsConfig

type JSONModelRepository struct {
	file string // path to models.json (e.g., .util/models.json)
}

func NewJSONModelRepository(file string) *JSONModelRepository {
	if file == "" {
		file = "src/.config/models.json"
	}
	return &JSONModelRepository{file: file}
}

func (r *JSONModelRepository) GetAll() ([]*types.Model, error) {
	data, err := os.ReadFile(r.file)
	if err != nil {
		return nil, err
	}
	var config types.ModelsConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	var models []*types.Model
	for i := range config.Models {
		models = append(models, &config.Models[i])
	}
	return models, nil
}

func (r *JSONModelRepository) GetByID(name string) (*types.Model, error) {
	models, err := r.GetAll()
	if err != nil {
		return nil, err
	}
	for _, m := range models {
		if m.Name == name {
			return m, nil
		}
	}
	return nil, os.ErrNotExist
}

func (r *JSONModelRepository) Save(model *types.Model) error {
	if model == nil || model.Name == "" {
		return os.ErrInvalid
	}
	data, err := os.ReadFile(r.file)
	var config types.ModelsConfig
	if err == nil {
		_ = json.Unmarshal(data, &config)
	}
	updated := false
	for i := range config.Models {
		if config.Models[i].Name == model.Name {
			config.Models[i] = *model
			updated = true
			break
		}
	}
	if !updated {
		config.Models = append(config.Models, *model)
	}
	out, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	tmp := r.file + ".tmp"
	if err := os.WriteFile(tmp, out, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, r.file)
}

func (r *JSONModelRepository) Delete(name string) error {
	data, err := os.ReadFile(r.file)
	if err != nil {
		return err
	}
	var config types.ModelsConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}
	var newModels []types.Model
	for _, m := range config.Models {
		if m.Name != name {
			newModels = append(newModels, m)
		}
	}
	config.Models = newModels
	out, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	tmp := r.file + ".tmp"
	if err := os.WriteFile(tmp, out, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, r.file)
}

// JSONChatService implements ChatService using a ChatRepository.
type JSONChatService struct {
	repo *JSONChatRepository
}

func NewJSONChatService(repo *JSONChatRepository) *JSONChatService {
	return &JSONChatService{repo: repo}
}

// ListChats returns a list of ChatInfo for all chats, including favorite and modification time.
func (s *JSONChatService) ListChats() ([]types.ChatInfo, error) {
	chats, err := s.repo.GetAll()
	if err != nil {
		return nil, err
	}
	var infos []types.ChatInfo
	for _, chat := range chats {
		fileInfo, ferr := s.repo.GetChatFileInfo(chat.Metadata.Title)
		modTime := int64(0)
		if ferr == nil {
			modTime = fileInfo.ModTime().Unix()
		}
		infos = append(infos, types.ChatInfo{
			Name:       chat.Metadata.Title,
			Favorite:   chat.Metadata.Favorite,
			ModifiedAt: modTime,
		})
	}
	return infos, nil
}

// GetChatMetadata returns the metadata for a chat by name.
func (s *JSONChatService) GetChatMetadata(name string) (*types.ChatMetadata, error) {
	chat, err := s.repo.GetByID(name)
	if err != nil {
		return nil, err
	}
	return &chat.Metadata, nil
}
