package localstorage

import (
	"sync"

	"go.mau.fi/whatsmeow"
)

const (
	StateIdle = iota
	StateWaitingScanQr
	StateWaitingFile
	StateWaitingInterval
	StateWaitingText
)

type UserInfo struct {
	State int
	WAClient *whatsmeow.Client
	MsgInterval int
	SentedFileID string
}

type Storage struct {
	mu *sync.Mutex
	smap map[int64]*UserInfo	
}

func New() *Storage {
	return &Storage{mu: &sync.Mutex{}, smap: make(map[int64]*UserInfo)}
}

func (s *Storage) Set(userID int64, userInfo *UserInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.smap[userID] = userInfo
}

func (s *Storage) Get(userID int64) (*UserInfo, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	userInfo, ok := s.smap[userID]
	return userInfo, ok
}

func (s *Storage) SetState(userID int64, state int) {
	if userInfo, ok := s.Get(userID); ok {
			userInfo.State = state
			s.Set(userID, userInfo)
	} else {
		userInfo = &UserInfo{State: state}
		s.Set(userID, userInfo)
	}
}

func (s *Storage) SetWAClient(userID int64, waClient *whatsmeow.Client) bool {
	userInfo, ok := s.Get(userID)
	if !ok {
		return false
	}
	
	userInfo.WAClient = waClient
	s.Set(userID, userInfo)
	return true

}

func (s *Storage) SetFileID(userID int64, fileID string) bool {
	userInfo, ok := s.Get(userID)
	if !ok {
		return false
	}
	
	userInfo.SentedFileID = fileID
	s.Set(userID, userInfo)
	return true
}

func (s *Storage) SetInterval(userID int64, interval int) bool {
	userInfo, ok := s.Get(userID)
	if !ok {
		return false
	}
	
	userInfo.MsgInterval = interval
	s.Set(userID, userInfo)
	return true	
}

func (s *Storage) Delete(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.smap, userID)
}
