package service

import "novel-be/internal/models"

type flowService struct {
	scene SceneService
}

func NewFlowService(scene SceneService) FlowService {
	return &flowService{scene: scene}
}

func (f *flowService) GetScene(sceneID int) (models.SceneResponse, error) {
	return f.scene.GetScene(sceneID)
}

func (f *flowService) GetWelcome() string {
	return "Welcome to Novel Reader API"
}
