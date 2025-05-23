package builtin

import (
	"sort"
	"strings"

	"github.com/threagile/threagile/pkg/types"
)

type MissingCloudHardeningRule struct{}

func NewMissingCloudHardeningRule() *MissingCloudHardeningRule {
	return &MissingCloudHardeningRule{}
}

func (*MissingCloudHardeningRule) Category() *types.RiskCategory {
	return &types.RiskCategory{
		ID:    "missing-cloud-hardening",
		Title: "Missing Cloud Hardening",
		Description: "Cloud components should be hardened according to the cloud vendor best practices. This affects their " +
			"configuration, auditing, and further areas.",
		Impact:     "If this risk is unmitigated, attackers might access cloud components in an unintended way.",
		ASVS:       "V1 - Architecture, Design and Threat Modeling Requirements",
		CheatSheet: "https://cheatsheetseries.owasp.org/cheatsheets/Attack_Surface_Analysis_Cheat_Sheet.html",
		Action:     "Cloud Hardening",
		Mitigation: "Apply hardening of all cloud components and services, taking special care to follow the individual risk descriptions (which " +
			"depend on the cloud provider tags in the model). " +
			"<br><br>For <b>Amazon Web Services (AWS)</b>: Follow the <i>CIS Benchmark for Amazon Web Services</i> (see also the automated checks of cloud audit tools like <i>\"PacBot\", \"CloudSploit\", \"CloudMapper\", \"ScoutSuite\", or \"Prowler AWS CIS Benchmark Tool\"</i>). " +
			"<br>For EC2 and other servers running Amazon Linux, follow the <i>CIS Benchmark for Amazon Linux</i> and switch to IMDSv2. " +
			"<br>For S3 buckets follow the <i>Security Best Practices for Amazon S3</i> at <a href=\"https://docs.aws.amazon.com/AmazonS3/latest/dev/security-best-practices.html\">https://docs.aws.amazon.com/AmazonS3/latest/dev/security-best-practices.html</a> to avoid accidental leakage. " +
			"<br>Also take a look at some of these tools: <a href=\"https://github.com/toniblyx/my-arsenal-of-aws-security-tools\">https://github.com/toniblyx/my-arsenal-of-aws-security-tools</a> " +
			"<br><br>For <b>Microsoft Azure</b>: Follow the <i>CIS Benchmark for Microsoft Azure</i> (see also the automated checks of cloud audit tools like <i>\"CloudSploit\" or \"ScoutSuite\"</i>)." +
			"<br><br>For <b>Google Cloud Platform</b>: Follow the <i>CIS Benchmark for Google Cloud Computing Platform</i> (see also the automated checks of cloud audit tools like <i>\"CloudSploit\" or \"ScoutSuite\"</i>). " +
			"<br><br>For <b>Oracle Cloud Platform</b>: Follow the hardening best practices (see also the automated checks of cloud audit tools like <i>\"CloudSploit\"</i>).",
		Check:          "Are recommendations from the linked cheat sheet and referenced ASVS chapter applied?",
		Function:       types.Operations,
		STRIDE:         types.Tampering,
		DetectionLogic: "In-scope cloud components (either residing in cloud trust boundaries or more specifically tagged with cloud provider types).",
		RiskAssessment: "The risk rating depends on the sensitivity of the technical asset itself and of the data assets processed.",
		FalsePositives: "Cloud components not running parts of the target architecture can be considered " +
			"as false positives after individual review.",
		ModelFailurePossibleReason: false,
		CWE:                        1008,
	}
}

var specificSubTagsAWS = []string{"aws:vpc", "aws:ec2", "aws:s3", "aws:ebs", "aws:apigateway", "aws:lambda", "aws:dynamodb", "aws:rds", "aws:sqs", "aws:iam"}

func (*MissingCloudHardeningRule) SupportedTags() []string {
	res := []string{
		"aws",   // Amazon AWS
		"azure", // Microsoft Azure
		"gcp",   // Google Cloud Platform
		"ocp",   // Oracle Cloud Platform
	}
	res = append(res, specificSubTagsAWS...)
	return res
}

func (r *MissingCloudHardeningRule) GenerateRisks(input *types.Model) ([]*types.Risk, error) {
	risks := make([]*types.Risk, 0)

	sharedRuntimesWithUnspecificCloudRisks := make(map[string]bool)
	trustBoundariesWithUnspecificCloudRisks := make(map[string]bool)
	techAssetsWithUnspecificCloudRisks := make(map[string]bool)

	sharedRuntimeIDsAWS := make(map[string]bool)
	trustBoundaryIDsAWS := make(map[string]bool)
	techAssetIDsAWS := make(map[string]bool)

	sharedRuntimeIDsAzure := make(map[string]bool)
	trustBoundaryIDsAzure := make(map[string]bool)
	techAssetIDsAzure := make(map[string]bool)

	sharedRuntimeIDsGCP := make(map[string]bool)
	trustBoundaryIDsGCP := make(map[string]bool)
	techAssetIDsGCP := make(map[string]bool)

	sharedRuntimeIDsOCP := make(map[string]bool)
	trustBoundaryIDsOCP := make(map[string]bool)
	techAssetIDsOCP := make(map[string]bool)

	techAssetIDsWithSubtagSpecificCloudRisks := make(map[string]bool)

	for _, trustBoundary := range input.TrustBoundaries {
		taggedOuterTB := trustBoundary.IsTaggedWithAny(r.SupportedTags()...) // false = generic cloud risks only // true = cloud-individual risks
		if !taggedOuterTB && !trustBoundary.Type.IsWithinCloud() {
			continue
		}

		r.addTrustBoundaryAccordingToBaseTag(trustBoundary, trustBoundariesWithUnspecificCloudRisks,
			trustBoundaryIDsAWS, trustBoundaryIDsAzure, trustBoundaryIDsGCP, trustBoundaryIDsOCP)
		for _, techAssetID := range input.RecursivelyAllTechnicalAssetIDsInside(trustBoundary) {
			tA := input.TechnicalAssets[techAssetID]
			if tA.IsTaggedWithAny(r.SupportedTags()...) {
				addAccordingToBaseTag(tA, tA.Tags,
					techAssetIDsWithSubtagSpecificCloudRisks,
					techAssetIDsAWS, techAssetIDsAzure, techAssetIDsGCP, techAssetIDsOCP)
				continue
			}

			if taggedOuterTB {
				addAccordingToBaseTag(tA, trustBoundary.Tags,
					techAssetIDsWithSubtagSpecificCloudRisks,
					techAssetIDsAWS, techAssetIDsAzure, techAssetIDsGCP, techAssetIDsOCP)
				continue
			}

			techAssetsWithUnspecificCloudRisks[techAssetID] = true
		}
	}

	// now loop over all technical assets, trust boundaries, and shared runtimes model-wide by tag
	for _, tA := range input.TechnicalAssetsTaggedWithAny(r.SupportedTags()...) {
		addAccordingToBaseTag(tA, tA.Tags,
			techAssetIDsWithSubtagSpecificCloudRisks,
			techAssetIDsAWS, techAssetIDsAzure, techAssetIDsGCP, techAssetIDsOCP)
	}
	for _, tB := range input.TrustBoundariesTaggedWithAny(r.SupportedTags()...) {
		for _, candidateID := range input.RecursivelyAllTechnicalAssetIDsInside(tB) {
			tA := input.TechnicalAssets[candidateID]
			if tA.IsTaggedWithAny(r.SupportedTags()...) {
				addAccordingToBaseTag(tA, tA.Tags,
					techAssetIDsWithSubtagSpecificCloudRisks,
					techAssetIDsAWS, techAssetIDsAzure, techAssetIDsGCP, techAssetIDsOCP)
			} else {
				addAccordingToBaseTag(tA, tB.Tags,
					techAssetIDsWithSubtagSpecificCloudRisks,
					techAssetIDsAWS, techAssetIDsAzure, techAssetIDsGCP, techAssetIDsOCP)
			}
		}
	}
	for _, sR := range input.SharedRuntimes {
		r.addSharedRuntimeAccordingToBaseTag(sR, sharedRuntimesWithUnspecificCloudRisks,
			sharedRuntimeIDsAWS, sharedRuntimeIDsAzure, sharedRuntimeIDsGCP, sharedRuntimeIDsOCP)
		for _, candidateID := range sR.TechnicalAssetsRunning {
			tA := input.TechnicalAssets[candidateID]
			addAccordingToBaseTag(tA, sR.Tags,
				techAssetIDsWithSubtagSpecificCloudRisks,
				techAssetIDsAWS, techAssetIDsAzure, techAssetIDsGCP, techAssetIDsOCP)
		}
	}

	// remove from sharedRuntimesWithUnspecificCloudRisks all specific tagged assets
	for id := range sharedRuntimeIDsAWS {
		delete(sharedRuntimesWithUnspecificCloudRisks, id)
	}
	for id := range sharedRuntimeIDsAzure {
		delete(sharedRuntimesWithUnspecificCloudRisks, id)
	}
	for id := range sharedRuntimeIDsGCP {
		delete(sharedRuntimesWithUnspecificCloudRisks, id)
	}
	for id := range sharedRuntimeIDsOCP {
		delete(sharedRuntimesWithUnspecificCloudRisks, id)
	}

	// remove from trustBoundariesWithUnspecificCloudRisks all specific tagged assets
	for id := range trustBoundaryIDsAWS {
		delete(trustBoundariesWithUnspecificCloudRisks, id)
	}
	for id := range trustBoundaryIDsAzure {
		delete(trustBoundariesWithUnspecificCloudRisks, id)
	}
	for id := range trustBoundaryIDsGCP {
		delete(trustBoundariesWithUnspecificCloudRisks, id)
	}
	for id := range trustBoundaryIDsOCP {
		delete(trustBoundariesWithUnspecificCloudRisks, id)
	}

	// remove from techAssetsWithUnspecificCloudRisks all specific tagged assets
	for techAssetID := range techAssetIDsWithSubtagSpecificCloudRisks {
		delete(techAssetsWithUnspecificCloudRisks, techAssetID)
	}
	for techAssetID := range techAssetIDsAWS {
		delete(techAssetsWithUnspecificCloudRisks, techAssetID)
	}
	for techAssetID := range techAssetIDsAzure {
		delete(techAssetsWithUnspecificCloudRisks, techAssetID)
	}
	for techAssetID := range techAssetIDsGCP {
		delete(techAssetsWithUnspecificCloudRisks, techAssetID)
	}
	for techAssetID := range techAssetIDsOCP {
		delete(techAssetsWithUnspecificCloudRisks, techAssetID)
	}

	// NOW ACTUALLY CREATE THE RISKS
	addedAWS, addedAzure, addedGCP, addedOCP := false, false, false, false

	// first try to add shared runtimes...
	for id := range sharedRuntimeIDsAWS {
		risks = append(risks, r.createRiskForSharedRuntime(input, input.SharedRuntimes[id], "AWS", "CIS Benchmark for AWS"))
		addedAWS = true
	}
	for id := range sharedRuntimeIDsAzure {
		risks = append(risks, r.createRiskForSharedRuntime(input, input.SharedRuntimes[id], "Azure", "CIS Benchmark for Microsoft Azure"))
		addedAzure = true
	}
	for id := range sharedRuntimeIDsGCP {
		risks = append(risks, r.createRiskForSharedRuntime(input, input.SharedRuntimes[id], "GCP", "CIS Benchmark for Google Cloud Computing Platform"))
		addedGCP = true
	}
	for id := range sharedRuntimeIDsOCP {
		risks = append(risks, r.createRiskForSharedRuntime(input, input.SharedRuntimes[id], "OCP", "Vendor Best Practices for Oracle Cloud Platform"))
		addedOCP = true
	}
	for id := range sharedRuntimesWithUnspecificCloudRisks {
		risks = append(risks, r.createRiskForSharedRuntime(input, input.SharedRuntimes[id], "", ""))
	}

	// ... followed by trust boundaries for the generic risks
	for id := range trustBoundaryIDsAWS {
		risks = append(risks, r.createRiskForTrustBoundary(input, input.TrustBoundaries[id], "AWS", "CIS Benchmark for AWS"))
		addedAWS = true
	}
	for id := range trustBoundaryIDsAzure {
		risks = append(risks, r.createRiskForTrustBoundary(input, input.TrustBoundaries[id], "Azure", "CIS Benchmark for Microsoft Azure"))
		addedAzure = true
	}
	for id := range trustBoundaryIDsGCP {
		risks = append(risks, r.createRiskForTrustBoundary(input, input.TrustBoundaries[id], "GCP", "CIS Benchmark for Google Cloud Computing Platform"))
		addedGCP = true
	}
	for id := range trustBoundaryIDsOCP {
		risks = append(risks, r.createRiskForTrustBoundary(input, input.TrustBoundaries[id], "OCP", "Vendor Best Practices for Oracle Cloud Platform"))
		addedOCP = true
	}
	for id := range trustBoundariesWithUnspecificCloudRisks {
		risks = append(risks, r.createRiskForTrustBoundary(input, input.TrustBoundaries[id], "", ""))
	}

	// just use the most sensitive asset as an example - to only create one general "AWS cloud hardening" risk, not many
	if !addedAWS {
		mostRelevantAsset := findMostSensitiveTechnicalAsset(input, techAssetIDsAWS)
		if mostRelevantAsset != nil {
			risks = append(risks, r.createRiskForTechnicalAsset(input, mostRelevantAsset, "AWS", "CIS Benchmark for AWS"))
			addedAWS = true
		}
	}
	// just use the most sensitive asset as an example - to only create one general "Azure cloud hardening" risk, not many
	if !addedAzure {
		mostRelevantAsset := findMostSensitiveTechnicalAsset(input, techAssetIDsAzure)
		if mostRelevantAsset != nil {
			risks = append(risks, r.createRiskForTechnicalAsset(input, mostRelevantAsset, "Azure", "CIS Benchmark for Microsoft Azure"))
			addedAzure = true
		}
	}
	// just use the most sensitive asset as an example - to only create one general "GCP cloud hardening" risk, not many
	if !addedGCP {
		mostRelevantAsset := findMostSensitiveTechnicalAsset(input, techAssetIDsGCP)
		if mostRelevantAsset != nil {
			risks = append(risks, r.createRiskForTechnicalAsset(input, mostRelevantAsset, "GCP", "CIS Benchmark for Google Cloud Computing Platform"))
			addedGCP = true
		}
	}
	// just use the most sensitive asset as an example - to only create one general "GCP cloud hardening" risk, not many
	if !addedOCP {
		mostRelevantAsset := findMostSensitiveTechnicalAsset(input, techAssetIDsOCP)
		if mostRelevantAsset != nil {
			risks = append(risks, r.createRiskForTechnicalAsset(input, mostRelevantAsset, "OCP", "Vendor Best Practices for Oracle Cloud Platform"))
			addedOCP = true
		}
	}

	// now also add all tech asset specific tag-specific risks, as they are specific to the asset anyway (therefore don't set added to true here)
	for id := range techAssetIDsWithSubtagSpecificCloudRisks {
		tA := input.TechnicalAssets[id]
		if isTechnicalAssetTaggedWithAnyTraversingUp(input, tA, "aws:ec2") {
			risks = append(risks, r.createRiskForTechnicalAsset(input, tA, "EC2", "CIS Benchmark for Amazon Linux"))
		}
		if isTechnicalAssetTaggedWithAnyTraversingUp(input, tA, "aws:s3") {
			risks = append(risks, r.createRiskForTechnicalAsset(input, tA, "S3", "Security Best Practices for AWS S3"))
		}
		// TODO add more tag-specific risks like also for aws:lambda etc. here
	}

	_ = addedAWS
	_ = addedAzure
	_ = addedGCP
	_ = addedOCP

	return risks, nil
}

// first use the tag(s) of the asset itself, then their trust boundaries (recursively up) and then their shared runtime
func isTechnicalAssetTaggedWithAnyTraversingUp(model *types.Model, ta *types.TechnicalAsset, tags ...string) bool {
	if containsCaseInsensitiveAny(ta.Tags, tags...) {
		return true
	}
	tbID := model.GetTechnicalAssetTrustBoundaryId(ta)
	if len(tbID) > 0 {
		if isTrustedBoundaryTaggedWithAnyTraversingUp(model, model.TrustBoundaries[tbID], tags...) {
			return true
		}
	}
	for _, sr := range model.SharedRuntimes {
		if contains(sr.TechnicalAssetsRunning, ta.Id) && sr.IsTaggedWithAny(tags...) {
			return true
		}
	}
	return false
}

func isTrustedBoundaryTaggedWithAnyTraversingUp(model *types.Model, tb *types.TrustBoundary, tags ...string) bool {
	if tb.IsTaggedWithAny(tags...) {
		return true
	}
	parentTb := model.FindParentTrustBoundary(tb)
	if parentTb != nil && isTrustedBoundaryTaggedWithAnyTraversingUp(model, parentTb, tags...) {
		return true
	}
	return false
}

func (r *MissingCloudHardeningRule) addTrustBoundaryAccordingToBaseTag(trustBoundary *types.TrustBoundary,
	trustBoundariesWithUnspecificCloudRisks map[string]bool,
	trustBoundaryIDsAWS map[string]bool,
	trustBoundaryIDsAzure map[string]bool,
	trustBoundaryIDsGCP map[string]bool,
	trustBoundaryIDsOCP map[string]bool) {
	if trustBoundary.IsTaggedWithAny(r.SupportedTags()...) {
		if isTaggedWithBaseTag(trustBoundary.Tags, "aws") {
			trustBoundaryIDsAWS[trustBoundary.Id] = true
		}
		if isTaggedWithBaseTag(trustBoundary.Tags, "azure") {
			trustBoundaryIDsAzure[trustBoundary.Id] = true
		}
		if isTaggedWithBaseTag(trustBoundary.Tags, "gcp") {
			trustBoundaryIDsGCP[trustBoundary.Id] = true
		}
		if isTaggedWithBaseTag(trustBoundary.Tags, "ocp") {
			trustBoundaryIDsOCP[trustBoundary.Id] = true
		}
	} else {
		trustBoundariesWithUnspecificCloudRisks[trustBoundary.Id] = true
	}
}

func (r *MissingCloudHardeningRule) addSharedRuntimeAccordingToBaseTag(sharedRuntime *types.SharedRuntime,
	sharedRuntimesWithUnspecificCloudRisks map[string]bool,
	sharedRuntimeIDsAWS map[string]bool,
	sharedRuntimeIDsAzure map[string]bool,
	sharedRuntimeIDsGCP map[string]bool,
	sharedRuntimeIDsOCP map[string]bool) {
	if sharedRuntime.IsTaggedWithAny(r.SupportedTags()...) {
		if isTaggedWithBaseTag(sharedRuntime.Tags, "aws") {
			sharedRuntimeIDsAWS[sharedRuntime.Id] = true
		}
		if isTaggedWithBaseTag(sharedRuntime.Tags, "azure") {
			sharedRuntimeIDsAzure[sharedRuntime.Id] = true
		}
		if isTaggedWithBaseTag(sharedRuntime.Tags, "gcp") {
			sharedRuntimeIDsGCP[sharedRuntime.Id] = true
		}
		if isTaggedWithBaseTag(sharedRuntime.Tags, "ocp") {
			sharedRuntimeIDsOCP[sharedRuntime.Id] = true
		}
	} else {
		sharedRuntimesWithUnspecificCloudRisks[sharedRuntime.Id] = true
	}
}

func addAccordingToBaseTag(techAsset *types.TechnicalAsset, tags []string,
	techAssetIDsWithTagSpecificCloudRisks map[string]bool,
	techAssetIDsAWS map[string]bool,
	techAssetIDsAzure map[string]bool,
	techAssetIDsGCP map[string]bool,
	techAssetIDsOCP map[string]bool) {
	if techAsset.IsTaggedWithAny(specificSubTagsAWS...) {
		techAssetIDsWithTagSpecificCloudRisks[techAsset.Id] = true
	}
	if isTaggedWithBaseTag(tags, "aws") {
		techAssetIDsAWS[techAsset.Id] = true
	}
	if isTaggedWithBaseTag(tags, "azure") {
		techAssetIDsAzure[techAsset.Id] = true
	}
	if isTaggedWithBaseTag(tags, "gcp") {
		techAssetIDsGCP[techAsset.Id] = true
	}
	if isTaggedWithBaseTag(tags, "ocp") {
		techAssetIDsOCP[techAsset.Id] = true
	}
}

func isTaggedWithBaseTag(tags []string, baseTag string) bool { // base tags are before the colon ":" like in "aws:ec2" it's "aws". The subtag is after the colon. Also, a pure "aws" tag matches the base tag "aws"
	baseTag = strings.ToLower(strings.TrimSpace(baseTag))
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == baseTag || strings.HasPrefix(tag, baseTag+":") {
			return true
		}
	}
	return false
}

func findMostSensitiveTechnicalAsset(input *types.Model, techAssets map[string]bool) *types.TechnicalAsset {
	var mostRelevantAsset *types.TechnicalAsset
	// as in Go ranging over map is random order, range over them in sorted (hence reproducible) way:
	keys := make([]string, 0, len(techAssets))
	for k := range techAssets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, id := range keys {
		tA := input.TechnicalAssets[id]
		if mostRelevantAsset == nil || tA.HighestSensitivityScore() > mostRelevantAsset.HighestSensitivityScore() {
			mostRelevantAsset = tA
		}
	}
	return mostRelevantAsset
}

func (r *MissingCloudHardeningRule) createRiskForSharedRuntime(input *types.Model, sharedRuntime *types.SharedRuntime, prefix, details string) *types.Risk {
	id := ""
	if len(prefix) > 0 {
		id = "@" + strings.ToLower(prefix)
		prefix = " (" + prefix + ")"
	}
	title := "<b>Missing Cloud Hardening" + prefix + "</b> risk at <b>" + sharedRuntime.Title + "</b>"
	if len(details) > 0 {
		title += ": <u>" + details + "</u>"
	}
	impact := types.MediumImpact
	confidentiality := input.FindSharedRuntimeHighestConfidentiality(sharedRuntime)
	integrity := input.FindSharedRuntimeHighestIntegrity(sharedRuntime)
	availability := input.FindSharedRuntimeHighestAvailability(sharedRuntime)
	if confidentiality >= types.Confidential || integrity >= types.Critical || availability >= types.Critical {
		impact = types.HighImpact
	}
	if confidentiality == types.StrictlyConfidential || integrity == types.MissionCritical || availability == types.MissionCritical {
		impact = types.VeryHighImpact
	}
	// create risk
	risk := &types.Risk{
		CategoryId:                  r.Category().ID,
		Severity:                    types.CalculateSeverity(types.Unlikely, impact),
		ExploitationLikelihood:      types.Unlikely,
		ExploitationImpact:          impact,
		Title:                       title,
		MostRelevantSharedRuntimeId: sharedRuntime.Id,
		DataBreachProbability:       types.Probable,
		DataBreachTechnicalAssetIDs: sharedRuntime.TechnicalAssetsRunning,
	}
	risk.SyntheticId = risk.CategoryId + "@" + sharedRuntime.Id + id
	return risk
}

func (r *MissingCloudHardeningRule) createRiskForTrustBoundary(parsedModel *types.Model, trustBoundary *types.TrustBoundary, prefix, details string) *types.Risk {
	id := ""
	if len(prefix) > 0 {
		id = "@" + strings.ToLower(prefix)
		prefix = " (" + prefix + ")"
	}
	title := "<b>Missing Cloud Hardening" + prefix + "</b> risk at <b>" + trustBoundary.Title + "</b>"
	if len(details) > 0 {
		title += ": <u>" + details + "</u>"
	}
	impact := types.MediumImpact
	confidentiality := parsedModel.FindTrustBoundaryHighestConfidentiality(trustBoundary)
	integrity := parsedModel.FindTrustBoundaryHighestIntegrity(trustBoundary)
	availability := parsedModel.FindTrustBoundaryHighestAvailability(trustBoundary)
	if confidentiality >= types.Confidential || integrity >= types.Critical || availability >= types.Critical {
		impact = types.HighImpact
	}
	if confidentiality == types.StrictlyConfidential || integrity == types.MissionCritical || availability == types.MissionCritical {
		impact = types.VeryHighImpact
	}
	// create risk
	risk := &types.Risk{
		CategoryId:                  r.Category().ID,
		Severity:                    types.CalculateSeverity(types.Unlikely, impact),
		ExploitationLikelihood:      types.Unlikely,
		ExploitationImpact:          impact,
		Title:                       title,
		MostRelevantTrustBoundaryId: trustBoundary.Id,
		DataBreachProbability:       types.Probable,
		DataBreachTechnicalAssetIDs: parsedModel.RecursivelyAllTechnicalAssetIDsInside(trustBoundary),
	}
	risk.SyntheticId = risk.CategoryId + "@" + trustBoundary.Id + id
	return risk
}

func (r *MissingCloudHardeningRule) createRiskForTechnicalAsset(parsedModel *types.Model, technicalAsset *types.TechnicalAsset, prefix, details string) *types.Risk {
	id := ""
	if len(prefix) > 0 {
		id = "@" + strings.ToLower(prefix)
		prefix = " (" + prefix + ")"
	}
	title := "<b>Missing Cloud Hardening" + prefix + "</b> risk at <b>" + technicalAsset.Title + "</b>"
	if len(details) > 0 {
		title += ": <u>" + details + "</u>"
	}
	impact := types.MediumImpact
	if parsedModel.HighestProcessedConfidentiality(technicalAsset) >= types.Confidential ||
		parsedModel.HighestProcessedIntegrity(technicalAsset) >= types.Critical ||
		parsedModel.HighestProcessedAvailability(technicalAsset) >= types.Critical {
		impact = types.HighImpact
	}
	if parsedModel.HighestProcessedConfidentiality(technicalAsset) == types.StrictlyConfidential ||
		parsedModel.HighestProcessedIntegrity(technicalAsset) == types.MissionCritical ||
		parsedModel.HighestProcessedAvailability(technicalAsset) == types.MissionCritical {
		impact = types.VeryHighImpact
	}
	// create risk
	risk := &types.Risk{
		CategoryId:                   r.Category().ID,
		Severity:                     types.CalculateSeverity(types.Unlikely, impact),
		ExploitationLikelihood:       types.Unlikely,
		ExploitationImpact:           impact,
		Title:                        title,
		MostRelevantTechnicalAssetId: technicalAsset.Id,
		DataBreachProbability:        types.Probable,
		DataBreachTechnicalAssetIDs:  []string{technicalAsset.Id},
	}
	risk.SyntheticId = risk.CategoryId + "@" + technicalAsset.Id + id
	return risk
}
