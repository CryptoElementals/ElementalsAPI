package battle

// MultiplierCalculator multiplier calculator
type MultiplierCalculator struct{}

// NewMultiplierCalculator create a new multiplier calculator
func NewMultiplierCalculator() *MultiplierCalculator {
	return &MultiplierCalculator{}
}

// CalculateMultiplierByLostHP calculate multiplier based on accumulated lost health
func (mc *MultiplierCalculator) CalculateMultiplierByLostHP(lostHP int) float64 {
	if lostHP <= 2000 {
		return 1.0
	}

	excessHP := lostHP - 2000
	bonusMultiplier := float64(excessHP) / 500.0
	newMultiplier := 1.0 + bonusMultiplier

	if newMultiplier > 9.0 {
		newMultiplier = 9.0
	}

	return newMultiplier
}
