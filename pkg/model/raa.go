package model

import (
	"sort"

	"github.com/threagile/threagile/pkg/types"
)

func applyRAA(input *types.Model, progressReporter types.ProgressReporter) string {
	progressReporter.Infof("Applying RAA calculation")

	for techAssetID, techAsset := range input.TechnicalAssets {
		aa := calculateAttackerAttractiveness(input, techAsset)
		aa += calculatePivotingNeighbourEffectAdjustment(input, techAsset)
		techAsset.RAA = calculateRelativeAttackerAttractiveness(input, aa)
		input.TechnicalAssets[techAssetID] = techAsset
	}
	// return intro text (for reporting etc., can be short summary-like)
	return "For each technical asset the <b>\"Relative Attacker Attractiveness\"</b> (RAA) value was calculated " +
		"in percent. The higher the RAA, the more interesting it is for an attacker to compromise the asset. The calculation algorithm takes " +
		"the sensitivity ratings and quantities of stored and processed data into account as well as the communication links of the " +
		"technical asset. Neighbouring assets to high-value RAA targets might receive an increase in their RAA value when they have " +
		"a communication link towards that target (\"Pivoting-Factor\").<br><br>The following lists all technical assets sorted by their " +
		"RAA value from highest (most attacker attractive) to lowest. This list can be used to prioritize on efforts relevant for the most " +
		"attacker-attractive technical assets:"
}

// set the concrete value in relation to the minimum and maximum of all
func calculateRelativeAttackerAttractiveness(input *types.Model, attractiveness float64) float64 {
	var attackerAttractivenessMinimum, attackerAttractivenessMaximum, spread float64 = 0, 0, 0
	if attackerAttractivenessMinimum == 0 || attackerAttractivenessMaximum == 0 {
		attackerAttractivenessMinimum, attackerAttractivenessMaximum = 9223372036854775807, -9223372036854775808
		// determine (only one time required) the min/max of all
		// first create them in memory (see the link replacement below for nested trust boundaries) - otherwise in Go ranging over map is random order
		// range over them in sorted (hence re-producible) way:
		keys := make([]string, 0)
		for k := range input.TechnicalAssets {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			techAsset := input.TechnicalAssets[key]
			if calculateAttackerAttractiveness(input, techAsset) > attackerAttractivenessMaximum {
				attackerAttractivenessMaximum = calculateAttackerAttractiveness(input, techAsset)
			}
			if calculateAttackerAttractiveness(input, techAsset) < attackerAttractivenessMinimum {
				attackerAttractivenessMinimum = calculateAttackerAttractiveness(input, techAsset)
			}
		}
		if !(attackerAttractivenessMinimum < attackerAttractivenessMaximum) {
			attackerAttractivenessMaximum = attackerAttractivenessMinimum + 1
		}
		spread = attackerAttractivenessMaximum - attackerAttractivenessMinimum
	}
	// calculate the percent value of the value within the defined min/max range
	value := attractiveness - attackerAttractivenessMinimum
	percent := value / spread * 100
	if percent <= 0 {
		percent = 1 // since 0 suggests no attacks at all
	}
	return percent
}

// increase the RAA (relative attacker attractiveness) by one third (1/3) of the delta to the highest outgoing neighbour (if positive delta)
func calculatePivotingNeighbourEffectAdjustment(input *types.Model, techAsset *types.TechnicalAsset) float64 {
	if techAsset.OutOfScope {
		return 0
	}
	adjustment := 0.0
	for _, commLink := range techAsset.CommunicationLinks {
		outgoingNeighbour := input.TechnicalAssets[commLink.TargetId]
		//if outgoingNeighbour.getTrustBoundary() == techAsset.getTrustBoundary() { // same trust boundary
		delta := calculateRelativeAttackerAttractiveness(input, calculateAttackerAttractiveness(input, outgoingNeighbour)) - calculateRelativeAttackerAttractiveness(input, calculateAttackerAttractiveness(input, techAsset))
		if delta > 0 {
			potentialIncrease := delta / 3
			//fmt.Println("Positive delta from", techAsset.ID, "to", outgoingNeighbour.ID, "is", delta, "yields to pivoting neighbour effect of an increase of", potentialIncrease)
			if potentialIncrease > adjustment {
				adjustment = potentialIncrease
			}
		}
		//}
	}
	return adjustment
}

// The sum of all CIAs of the asset itself (fibonacci scale) plus the sum of the comm-links' transferred CIAs
// Multiplied by the quantity values of the data asset for C and I (not A)
func calculateAttackerAttractiveness(input *types.Model, techAsset *types.TechnicalAsset) float64 {
	if techAsset.OutOfScope {
		return 0
	}
	var score = 0.0
	score += techAsset.Confidentiality.AttackerAttractivenessForAsset()
	score += techAsset.Integrity.AttackerAttractivenessForAsset()
	score += techAsset.Availability.AttackerAttractivenessForAsset()
	for _, dataAssetProcessed := range techAsset.DataAssetsProcessed {
		dataAsset := input.DataAssets[dataAssetProcessed]
		score += dataAsset.Confidentiality.AttackerAttractivenessForProcessedOrStoredData() * dataAsset.Quantity.QuantityFactor()
		score += dataAsset.Integrity.AttackerAttractivenessForProcessedOrStoredData() * dataAsset.Quantity.QuantityFactor()
		score += dataAsset.Availability.AttackerAttractivenessForProcessedOrStoredData()
	}
	// NOTE: Assuming all stored data is also processed, this effectively scores stored data twice
	for _, dataAssetStored := range techAsset.DataAssetsStored {
		dataAsset := input.DataAssets[dataAssetStored]
		score += dataAsset.Confidentiality.AttackerAttractivenessForProcessedOrStoredData() * dataAsset.Quantity.QuantityFactor()
		score += dataAsset.Integrity.AttackerAttractivenessForProcessedOrStoredData() * dataAsset.Quantity.QuantityFactor()
		score += dataAsset.Availability.AttackerAttractivenessForProcessedOrStoredData()
	}
	// NOTE: To send or receive data effectively is processing that data and it's questionable if the attractiveness increases further
	for _, dataFlow := range techAsset.CommunicationLinks {
		for _, dataAssetSent := range dataFlow.DataAssetsSent {
			dataAsset := input.DataAssets[dataAssetSent]
			score += dataAsset.Confidentiality.AttackerAttractivenessForInOutTransferredData() * dataAsset.Quantity.QuantityFactor()
			score += dataAsset.Integrity.AttackerAttractivenessForInOutTransferredData() * dataAsset.Quantity.QuantityFactor()
			score += dataAsset.Availability.AttackerAttractivenessForInOutTransferredData()
		}

		for _, dataAssetReceived := range dataFlow.DataAssetsReceived {
			dataAsset := input.DataAssets[dataAssetReceived]
			score += dataAsset.Confidentiality.AttackerAttractivenessForInOutTransferredData() * dataAsset.Quantity.QuantityFactor()
			score += dataAsset.Integrity.AttackerAttractivenessForInOutTransferredData() * dataAsset.Quantity.QuantityFactor()
			score += dataAsset.Availability.AttackerAttractivenessForInOutTransferredData()
		}
	}

	if techAsset.Technologies.GetAttribute(types.LoadBalancer, types.ReverseProxy) {
		score = score / 5.5
	} else if techAsset.Technologies.GetAttribute(types.Monitoring) {
		score = score / 5
	} else if techAsset.Technologies.GetAttribute(types.ContainerPlatform) {
		score = score * 5
	} else if techAsset.Technologies.GetAttribute(types.Vault) {
		score = score * 2
	} else if techAsset.Technologies.GetAttribute(types.BuildPipeline, types.SourcecodeRepository, types.ArtifactRegistry) {
		score = score * 2
	} else if techAsset.Technologies.GetAttribute(types.IdentityProvider, types.IdentityStoreDatabase, types.IdentityStoreLDAP) {
		score = score * 2.5
	} else if techAsset.Type == types.Datastore {
		score = score * 2
	}

	if techAsset.MultiTenant {
		score = score * 1.5
	}

	return score
}
