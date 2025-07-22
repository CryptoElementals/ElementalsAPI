package battle

// MultiplierCalculator multiplier calculator
type MultiplierCalculator struct{}

// NewMultiplierCalculator create a new multiplier calculator
func NewMultiplierCalculator() *MultiplierCalculator {
	return &MultiplierCalculator{}
}

// CalculateMultiplierByLostHP calculate multiplier based on accumulated lost health
func (mc *MultiplierCalculator) CalculateMultiplierByLostHP(lostHP int) uint32 {
	if lostHP <= 2000 {
		return 1
	}

	excessHP := lostHP - 2000
	bonusMultiplier := uint32(excessHP) / 500
	newMultiplier := 1 + bonusMultiplier

	if newMultiplier > 9 {
		newMultiplier = 9
	}

	return newMultiplier
}
