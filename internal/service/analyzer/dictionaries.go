package analyzer

type DefaultDictionariesCreator struct {
}

type Dictionaries struct {
	PrefixDictionary      []string
	PrefixDictionaryIC    []string
	VerbsDictionary       []string
	PSKDictionary         []string
	ErrandBodyDictionary  []string
	RegionsAllias         map[string]string
	ExceptionsDictonary   []string
	ErrandTypesDictionary []string
}

func NewDictionariesCreator() DictionariesCreator {
	return &DefaultDictionariesCreator{}
}

func (c *DefaultDictionariesCreator) CreateDictionaries() *Dictionaries {
	return &Dictionaries{
		PrefixDictionary:      prefix,
		PrefixDictionaryIC:    prefixIC,
		VerbsDictionary:       verbs,
		PSKDictionary:         psk,
		ErrandBodyDictionary:  errandBody,
		RegionsAllias:         regionsMap,
		ExceptionsDictonary:   exceptions,
		ErrandTypesDictionary: types,
	}
}
