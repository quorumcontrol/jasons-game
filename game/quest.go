package game

import (
	"strings"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"

	"github.com/quorumcontrol/jasons-game/messages"
)

var questPath = []string{"tree", "jasons-game", "quest-log"}

type QuestCompletion struct {
	Finished bool
	Success  bool
}

type QuestStep struct {
	Index        int
	NewChainTree *consensus.SignedChainTree
	Message      *messages.QuestMessage
}

type Quest interface {
	ID() string                                                      // unique identifier of the quest for the ChainTree quest-log
	PrettyString() string                                            // name of the quest
	Start(game *Game) (bool, error)                                  // called every time the player does basically anything, so make it fast
	InitState(state *QuestState)                                     // initialize quest state
	State() *QuestState                                              // returns a pointer to the current quest state
	FirstStep() *QuestStep                                           // called when quest is started
	NextStep(actorCtx actor.Context, game *Game) (*QuestStep, error) // called every time the player does something, but only after the quest has started
	End(game *Game) (*QuestCompletion, error)                        // called every time Update is called, but if Finished is true will end the quest
}

type QuestState struct {
	started      bool
	currentStep  *QuestStep
	highestIndex int
	completed    bool
}

func updateQuests(actorCtx actor.Context, game *Game) {
	player := game.PlayerTree()

	for _, quest := range game.Quests() {
		if quest.State().started {
			completion, err := quest.End(game)
			if err != nil {
				log.Errorf("error checking end to %s quest: %v", quest, err)
			}

			if completion.Finished {
				quest.State().completed = true
			}

			questLogPath := strings.Join(append(questPath, quest.ID(), "completion"), "/")
			finishedChainTree, err := game.network.UpdateChainTree(player.tree, questLogPath, completion)
			if err != nil {
				log.Errorf("error adding quest completion to player chaintree: %v", err)
			}
			player.SetChainTree(finishedChainTree)

			step, err := quest.NextStep(actorCtx, game)
			if err != nil {
				log.Errorf("error updating %s quest: %v", quest, err)
			}

			if step.NewChainTree != nil {
				player.SetChainTree(step.NewChainTree)
			}

			if step.Index > quest.State().highestIndex {
				quest.State().highestIndex = step.Index
			}

			quest.State().currentStep = step
		} else if !quest.State().completed {
			start, err := quest.Start(game)
			if err != nil {
				log.Errorf("error checking for start to %s quest: %v", quest, err)
			}

			if start {
				quest.State().started = true
				firstStep := quest.FirstStep()
				quest.State().currentStep = firstStep
				if firstStep.NewChainTree != nil {
					player.SetChainTree(firstStep.NewChainTree)
				}
			}
		}
	}
}

func messageStep(i int, msg string) *QuestStep {
	return &QuestStep{
		Index: i,
		Message: &messages.QuestMessage{
			Message: msg,
		},
	}
}
