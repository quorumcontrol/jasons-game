package game

import (
	"fmt"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
)

/*
 * This is the first quest that introduces the player to basic navigation in their world, portals, ink, etc.
 */

type MeetTheWizard struct {
	state *QuestState
}

var _ Quest = &MeetTheWizard{}

func (q *MeetTheWizard) PrettyString() string {
	return "Meet The Wizard"
}

func (q *MeetTheWizard) Start(game *Game) (bool, error) {
	location := game.PlayerLocation()
	return location.IsOrigin(), nil
}

func (q *MeetTheWizard) InitState(state *QuestState) {
	q.state = state
}

func (q *MeetTheWizard) State() *QuestState {
	return q.state
}

func (q *MeetTheWizard) FirstStep() *QuestStep {
	return messageStep(0, `
The world swims in around you in distorted waves. It's disorienting and trying to recover yourself you close your eyes and check in with your other senses. 
You smell mustiness and torches and a dash of plain old rotten.
You hear a voice... an old... and clearly impatient voice.
"Get up, get up.  We don't have time for this.  We need to find some ink and get out of here.
I have things to do," the voice wheezes.
Your eyes are cooperating now and you see before you a short, rotund gentlemen with a strange, star-covered vest and a ridiculous pointy hat.  He is staring at you with eyes deep-set and intensely green.
He repeats in his deep, and deeply irritated tone "C'mon, c'mon. Follow me north. The rats must have made off with my heptahydrate."
You see his rotund backside and he swirls his robe dramatically as he turns and heads North leaving you quite alone.
`)
}

func (q *MeetTheWizard) secondStep() *QuestStep {
	return messageStep(1, `
You follow the old man as soon as your feet allow it and rush to catch up with him.  
You pass through a corridor with a door to the left and right but he is headed to the door straight ahead.
He pushes through it muttering about "portals to nowhere and no-good layabouts indeed."
`)
}

func (q *MeetTheWizard) thirdStep() *QuestStep {
	return messageStep(2, `
Finally catching up in the room beyond he has turned on his heels and is pointing to his left at a gaping dark hole in the wall.
"C'mon now, get in there and track down my coperas. It is all I am missing to create some ink and get us out of this dead end dimension.  
"Unless you have some Ink in which case why in tarnation did you not tell me before!?"
"Yes, yes I thought not. You don't look the sort to know what's going on!"
"I will certainly not fit through that hole but you can!"
You are not sure if it's comforting or disturbing (perhaps it's both) but you do hear the squeaking of rats in the chamber beyond.
`)
}

func (q *MeetTheWizard) fourthStep(actorCtx actor.Context, game *Game) *QuestStep {
	currentLocation := game.PlayerLocation()

	response, err := actorCtx.RequestFuture(game.inventory, &CreateObjectRequest{
		Owner:       currentLocation,
		Name:        "green-vitriol",
		Loc:         []int64{currentLocation.X, currentLocation.Y},
		Description: "Is this what the wizard is looking for?",
	}, 5*time.Second).Result()
	if err != nil {
		log.Errorf("error creating quest object: %v", err)
		return nil
	}
	objCreateResponse, ok := response.(*CreateObjectResponse)
	if !ok {
		log.Errorf("error casting create object response; is of type %T", response)
		return nil
	}

	if objCreateResponse.Error != nil {
		log.Errorf("error creating quest object: %v", objCreateResponse.Error)
		return nil
	}

	return messageStep(3, `
You crawl into the hole cautiously on hands and knees.  
The muck slows you down but a good push from your friend from behind slides you forward through it.
The smell of mustiness turns even more putrid and you gag but keep down whatever is in your stomach.
A small dog like shape seems to rear up in front of you lit from behind. Small dog, extremely large rat more like it.
As your eyes adjust you are in some sort of small space but thankfully the rats seem to have decided they had better places to be.
You think to yourself you could not agree more.
The rats nest is made up of piles of old rags though not much of value. 
From further east you see another opening in the wall and faintly hear a familiar (if grating) voice... "Head this way once you have it..."
`)
}

func (q *MeetTheWizard) fifthStep() *QuestStep {
	return messageStep(4, `
You squeeze out of the hole with some difficulty into a more cheery room brightly lit with torches.
Your friend from earlier is staring... 
"WELL, do you HAVE it?"
"If not we are well and truly stuck... place it right here if you have it!"
"If not get back in there and look some more... it must be in there somewhere."
`)
}

func (q *MeetTheWizard) sixthStep() *QuestStep {
	return messageStep(5, `
"Ahh perfect!" The ornery fellow seems to brighten when he sees the items you have assembled.  
Strange tools flash from the pockets of his vest and quicker than you can comprehend the process is done.
He cackles gleefully and hands you a vial of ink.
"This is for your help!"
"This I need to create a portal back to civilization."
He spins his short little arms after dunking his fingers into an inkwell much like the one he gave to you.
"Over and out!" he cackles to himself.
A glowing circle the size of a man comes into view dazzlingly bright and somehow jet-black like the ink at the same time.  
The strange fellow completes his arm twirl with a flourish.
"Not entirely sure how I got stuck in your world here... but you might want to spruce up the place."
"See you around!"
With that he jumps into the glow disk and is gone.
`)
}

func (q *MeetTheWizard) NextStep(actorCtx actor.Context, game *Game) (*QuestStep, error) {
	location, err := game.refreshLocation()
	if err != nil {
		return nil, fmt.Errorf("error refreshing location: %v", err)
	}

	switch {
	case location.X == 0 && location.Y == 1 && q.state.highestIndex == 0:
		return q.secondStep(), nil
	case location.X == 0 && location.Y == 2 && q.state.highestIndex == 1:
		return q.thirdStep(), nil
	case location.X == 1 && location.Y == 2 && q.state.highestIndex == 2:
		return q.fourthStep(actorCtx, game), nil
	case location.X == 0 && location.Y == 2 && q.state.highestIndex == 3:
		return q.fifthStep(), nil
	case location.X == 0 && location.Y == 2 && q.state.highestIndex == 4:
		_, droppedGreenVitriol := location.Inventory["green-vitriol"]
		if droppedGreenVitriol {
			return q.sixthStep(), nil
		} else {
			time.Sleep(1 * time.Second)
			return q.NextStep(actorCtx, game)
		}
	// TODO: Finish rest of the steps
	default:
		return messageStep(-1, "You are off the beaten path."), nil
	}
}

func (q *MeetTheWizard) End(game *Game) (*QuestCompletion, error) {
	// TODO: Write the real end conditions
	return &QuestCompletion{
		Finished: false,
	}, nil
}
