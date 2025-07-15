package battle

import "fmt"

// BattleEngine battle engine
type BattleEngine struct {
	cardFactory      *CardFactory
	elementalSystem  *ElementalSystem
	multiplierCalc   *MultiplierCalculator
	gameLogic        *GameLogic
	rewardCalculator *RewardCalculator
}

// NewBattleEngine create a new battle engine
func NewBattleEngine() *BattleEngine {
	return &BattleEngine{
		cardFactory:      NewCardFactory(),
		elementalSystem:  NewElementalSystem(),
		multiplierCalc:   NewMultiplierCalculator(),
		gameLogic:        NewGameLogic(),
		rewardCalculator: NewRewardCalculator(),
	}
}

// ExecuteBattle execute battle
func (be *BattleEngine) ExecuteBattle(input *BattleInput, stage int) (*BattleResult, error) {
	if err := be.gameLogic.ValidateBattleInput(input); err != nil {
		return nil, err
	}

	if stage < 1 || stage > 3 {
		return nil, fmt.Errorf("stage parameter must be between 1 and 3")
	}

	player1Cards, err := be.cardFactory.GetCards(input.Player1Cards)
	if err != nil {
		return nil, err
	}
	player2Cards, err := be.cardFactory.GetCards(input.Player2Cards)
	if err != nil {
		return nil, err
	}

	result := &BattleResult{
		Player1Address: input.Player1Address,
		Player2Address: input.Player2Address,
		Stage:          stage,
		Rounds:         make([]RoundResult, 0),
	}

	currentPlayer1HP := input.Player1HP
	currentPlayer2HP := input.Player2HP
	currentPlayer1Multiplier := input.Player1Multiplier
	currentPlayer2Multiplier := input.Player2Multiplier

	player1LostHP := input.Player1LostHP
	player2LostHP := input.Player2LostHP

	for round := 0; round < 3; round++ {
		player1Card := player1Cards[round]
		player2Card := player2Cards[round]

		relation := be.elementalSystem.GetElementalRelation(player1Card, player2Card)
		actions := be.elementalSystem.BuildActions(player1Card, player2Card, relation)
		player1Damage, player2Damage := be.elementalSystem.ExecuteActions(actions)

		currentPlayer1HP += player1Damage
		currentPlayer2HP += player2Damage

		if currentPlayer1HP < 0 {
			currentPlayer1HP = 0
		}
		if currentPlayer2HP < 0 {
			currentPlayer2HP = 0
		}

		player1LostHP = input.Player1HP - currentPlayer1HP
		player2LostHP = input.Player2HP - currentPlayer2HP

		currentPlayer1Multiplier = be.multiplierCalc.CalculateMultiplierByLostHP(player1LostHP)
		currentPlayer2Multiplier = be.multiplierCalc.CalculateMultiplierByLostHP(player2LostHP)

		roundResult := RoundResult{
			RoundNumber:            round + 1,
			Player1CardID:          player1Card.ID,
			Player2CardID:          player2Card.ID,
			RelationType:           relation.Type,
			Actions:                actions,
			Player1HPDelta:         player1Damage,
			Player2HPDelta:         player2Damage,
			Player1HPAfter:         currentPlayer1HP,
			Player2HPAfter:         currentPlayer2HP,
			Player1MultiplierAfter: currentPlayer1Multiplier,
			Player2MultiplierAfter: currentPlayer2Multiplier,
			Description:            relation.Description,
		}

		result.Rounds = append(result.Rounds, roundResult)

		if isGameOver, winner := be.gameLogic.CheckGameOver(currentPlayer1HP, currentPlayer2HP, input.Player1Address, input.Player2Address); isGameOver {
			result.IsGameOver = true
			result.Winner = winner
			result.GameResultType = be.determineGameResultType(currentPlayer1HP, currentPlayer2HP)

			result.Player1FinalHP = currentPlayer1HP
			result.Player2FinalHP = currentPlayer2HP
			result.Player1LostHP = player1LostHP
			result.Player2LostHP = player2LostHP
			result.Player1FinalMultiplier = currentPlayer1Multiplier
			result.Player2FinalMultiplier = currentPlayer2Multiplier

			if winner == input.Player1Address {
				result.GameFinalMultiplier = currentPlayer2Multiplier
			} else if winner == input.Player2Address {
				result.GameFinalMultiplier = currentPlayer1Multiplier
			} else {
				result.GameFinalMultiplier = 1.0
			}

			result.Reward = be.rewardCalculator.CalculateRewards(result)
			return result, nil
		}
	}

	result.Player1FinalHP = currentPlayer1HP
	result.Player2FinalHP = currentPlayer2HP
	result.Player1LostHP = player1LostHP
	result.Player2LostHP = player2LostHP
	result.Player1FinalMultiplier = currentPlayer1Multiplier
	result.Player2FinalMultiplier = currentPlayer2Multiplier

	if stage == 3 {
		result.IsGameOver = true
		result.Winner = be.gameLogic.GetWinner(currentPlayer1HP, currentPlayer2HP, input.Player1Address, input.Player2Address)
		result.GameResultType = be.determineGameResultType(currentPlayer1HP, currentPlayer2HP)

		if result.Winner == input.Player1Address {
			result.GameFinalMultiplier = currentPlayer2Multiplier
		} else if result.Winner == input.Player2Address {
			result.GameFinalMultiplier = currentPlayer1Multiplier
		} else {
			result.GameFinalMultiplier = 1.0
		}

		result.Reward = be.rewardCalculator.CalculateRewards(result)
	} else {
		result.IsGameOver = false
		result.Winner = ""
		result.GameResultType = ""
		result.Reward = nil
	}

	return result, nil
}

// determineGameResultType determine game result type
func (be *BattleEngine) determineGameResultType(player1HP, player2HP int) string {
	if player1HP == player2HP {
		return "tie"
	} else if player1HP <= 0 || player2HP <= 0 {
		return "ko"
	} else {
		return "normal"
	}
}
