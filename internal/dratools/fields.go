package dratools

const (
	Name    = "dratools"
	Version = "0.0.1"

	SRARunResourceType        = "sra-run"
	SRAExperimentResourceType = "sra-experiment"
	SRASampleResourceType     = "sra-sample"
	SRAStudyResourceType      = "sra-study"
	SRASubmissionResourceType = "sra-submission"
	BioProjectResourceType    = "bioproject"
	BioSampleResourceType     = "biosample"

	FileTypeSRA   = "sra"
	FileTypeFASTQ = "fastq"
	FileTypeAll   = "all"

	DBXrefsKey           = "dbXrefs"
	ChildBioProjectsKey  = "childBioProjects"
	TypeKey              = "type"
	IDKey                = "id"
	IdentifierKey        = "identifier"
	AccessionKey         = "accession"
	PrimaryIDKey         = "primaryId"
	DistributionKey      = "distribution"
	ContentURLKey        = "contentUrl"
	ContentSizeKey       = "contentSize"
	MD5Key               = "md5"
	MD5SumKey            = "md5sum"
	EncodingFormatKey    = "encodingFormat"
	defaultDDBJSearchURL = "https://ddbj.nig.ac.jp/search/api"
)

var infoFieldKeys = []string{
	IdentifierKey,
	TypeKey,
	"title",
	"description",
	"organism",
	"platform",
	"instrumentModel",
	"libraryStrategy",
	"librarySource",
	"librarySelection",
	"libraryLayout",
	"libraryName",
	"dateCreated",
	"dateModified",
	"datePublished",
	"status",
}
