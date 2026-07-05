package dratools

import "regexp"

type accessionRule struct {
	pattern *regexp.Regexp
	typ     string
}

var accessionRules = []accessionRule{
	{regexp.MustCompile(`^[DES]RR\d+$`), SRARunResourceType},
	{regexp.MustCompile(`^[DES]RX\d+$`), SRAExperimentResourceType},
	{regexp.MustCompile(`^[DES]RS\d+$`), SRASampleResourceType},
	{regexp.MustCompile(`^[DES]RP\d+$`), SRAStudyResourceType},
	{regexp.MustCompile(`^[DES]RA\d+$`), SRASubmissionResourceType},
	{regexp.MustCompile(`^PRJ[DEN][A-Z]\d+$`), BioProjectResourceType},
	{regexp.MustCompile(`^SAM(?:D|N|EA|EG)?\d+$`), BioSampleResourceType},
}

func ResourceTypeFor(accession string) (string, error) {
	for _, rule := range accessionRules {
		if rule.pattern.MatchString(accession) {
			return rule.typ, nil
		}
	}
	return "", newError("unsupported_accession", "unsupported accession: "+accession)
}
