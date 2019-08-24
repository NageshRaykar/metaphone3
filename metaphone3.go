package metaphone3

import (
	"unicode"
)

// DefaultMaxLength is the max number of runes in a result when not specified in the encoder
var DefaultMaxLength = 8

// Encoder is a metaphone3 encoder that contains options and state for encoding.  It is not
// safe to use across goroutines.
type Encoder struct {
	// EncodeVowels determines if Metaphone3 will encode non-initial vowels. However, even
	// if there are more than one vowel sound in a vowel sequence (i.e.
	// vowel diphthong, etc.), only one 'A' will be encoded before the next consonant or the
	// end of the word.
	EncodeVowels bool

	// EncodeExact controls if Metaphone3 will encode consonants as exactly as possible.
	// This does not include 'S' vs. 'Z', since americans will pronounce 'S' at the
	// at the end of many words as 'Z', nor does it include "CH" vs. "SH". It does cause
	// a distinction to be made between 'B' and 'P', 'D' and 'T', 'G' and 'K', and 'V'
	// and 'F'.
	EncodeExact bool

	// The max allowed length of the output metaphs, if <= 0 then the DefaultMaxLength is used
	MaxLength int

	in                 []rune
	idx                int
	lastIdx            int
	primBuf, secondBuf []rune
	flagAlInversion    bool
}

// Encode takes in a string and returns primary and secondary metaphones.
// Both will be blank if given a blank input, and secondary can be blank
// if there's only one metaphone.
func (e *Encoder) Encode(in string) (primary, secondary string) {
	if in == "" {
		return "", ""
	}

	if e.MaxLength <= 0 {
		e.MaxLength = DefaultMaxLength
	}

	e.flagAlInversion = false

	// setup our input buffer and to-upper everything
	e.in = make([]rune, 0, len(in))
	for _, r := range in {
		e.in = append(e.in, unicode.ToUpper(r))
	}
	e.lastIdx = len(e.in) - 1

	e.primBuf = primeBuf(e.primBuf, e.MaxLength)
	e.secondBuf = primeBuf(e.secondBuf, e.MaxLength)

	// lets go rune-by-rune through the input string
	for e.idx = 0; e.idx < len(e.in); e.idx++ {

		// double check our output buffers, if they're full then we're done
		if len(e.primBuf) >= e.MaxLength || len(e.secondBuf) >= e.MaxLength {
			break
		}

		switch c := e.in[e.idx]; c {
		case 'B':
			e.encodeB()
		case 'ß', 'Ç':
			e.metaphAdd('S')
		case 'C':
			e.encodeC()
		case 'D':
			e.encodeD()
		case 'F':
			e.encodeF()
		case 'G':
			e.encodeG()
		case 'H':
			e.encodeH()
		case 'J':
			e.encodeJ()
		case 'K':
			e.encodeK()
		case 'L':
			e.encodeL()
		case 'M':
			e.encodeM()
		case 'N':
			e.encodeN()
		case 'Ñ':
			e.metaphAdd('N')
		case 'P':
			e.encodeP()
		case 'Q':
			e.encodeQ()
		case 'R':
			e.encodeR()
		case 'S':
			e.encodeS()
		case 'T':
			e.encodeT()
		case 'Ð', 'Þ':
			e.metaphAdd('0')
		case 'V':
			e.encodeV()
		case 'W':
			e.encodeW()
		case 'X':
			e.encodeX()
		case '\uC28A':
			//wat?
			e.metaphAdd('X')
		case '\uC28E':
			//wat?
			e.metaphAdd('S')
		case 'Z':
			e.encodeZ()
		default:
			if isVowel(c) {
				e.encodeVowels()
			}
		}
	}

	if areEqual(e.primBuf, e.secondBuf) {
		return string(e.primBuf), ""
	}

	return string(e.primBuf), string(e.secondBuf)
}

//////////////////////////////////////////////////////////////////////////////////////////////////////
// Detailed encoder functions
//////////////////////////////////////////////////////////////////////////////////////////////////////

func (e *Encoder) encodeB() {
	if e.encodeSilentB() {
		return
	}

	// "-mb", e.g", "dumb", already skipped over under
	// 'M', altho it should really be handled here...
	e.metaphAddExactApprox("B", "P")

	// skip double B, or BPx where X isn't H
	if e.charNextIs('B') ||
		(e.charNextIs('P') && e.idx+2 < len(e.in) && e.in[e.idx+2] != 'H') {
		e.idx++
	}
}

// Encodes silent 'B' for cases not covered under "-mb-"
func (e *Encoder) encodeSilentB() bool {
	//'debt', 'doubt', 'subtle'
	if e.stringAt(-2, "DEBT", "SUBTL", "SUBTIL") || e.stringAt(-3, "DOUBT") {
		e.metaphAdd('T')
		e.idx++
		return true
	}

	return false
}

func (e *Encoder) encodeC() {
	if e.encodeSilentCAtBeginning() ||
		e.encodeCaToS() ||
		e.encodeCoToS() ||
		e.encodeCh() ||
		e.encodeCcia() ||
		e.encodeCc() ||
		e.encodeCkCgCq() ||
		e.encodeCFrontVowel() ||
		e.encodeSilentC() ||
		e.encodeCz() ||
		e.encodeCs() {
		return
	}

	if e.stringAt(-1, "C", "K", "G", "Q") {
		e.metaphAdd('K')
	}

	//name sent in 'mac caffrey', 'mac gregor
	if e.stringAt(1, " C", " Q", " G") {
		e.idx++
	} else {
		if e.stringAt(1, "C", "K", "Q") && !e.stringAt(1, "CE", "CI") {
			e.idx++
			// account for combinations such as Ro-ckc-liffe
			if e.stringAt(0, "C", "K", "Q") && !e.stringAt(1, "CE", "CI") {
				e.idx++
			}
		}
	}
}

func (e *Encoder) encodeSilentCAtBeginning() bool {
	if e.idx == 0 && e.stringAt(0, "CT", "CN") {
		return true
	}
	return false
}

//Encodes exceptions where "-CA-" should encode to S
//instead of K including cases where the cedilla has not been used
func (e *Encoder) encodeCaToS() bool {
	// Special case: 'caesar'.
	// Also, where cedilla not used, as in "linguica" => LNKS
	if (e.idx == 0 && e.stringAt(0, "CAES", "CAEC", "CAEM")) ||
		e.stringStart("FACADE", "FRANCAIS", "FRANCAIX", "LINGUICA", "GONCALVES", "PROVENCAL") {
		e.metaphAdd('S')
		e.advanceCounter(1, 0)
		return true
	}

	return false
}

//Encodes exceptions where "-CO-" encodes to S instead of K
//including cases where the cedilla has not been used
func (e *Encoder) encodeCoToS() bool {
	// e.g. 'coelecanth' => SLKN0
	if e.stringAt(0, "COEL") && (e.isVowelAt(4) || e.idx+3 == e.lastIdx) ||
		e.stringAt(0, "COENA", "COENO") || e.stringStart("GARCON", "FRANCOIS", "MELANCON") {

		e.metaphAdd('S')
		e.advanceCounter(2, 0)
		return true
	}

	return false
}

func (e *Encoder) encodeCh() bool {
	if !e.stringAt(0, "CH") {
		return false
	}

	if e.encodeChae() ||
		e.encodeChToH() ||
		e.encodeSilentCh() ||
		e.encodeArch() ||
		e.encodeChToX() ||
		e.encodeEnglishChToK() ||
		e.encodeGermanicChToK() ||
		e.encodeGreekChInitial() ||
		e.encodeGreekChNonInitial() {
		return true
	}

	if e.idx > 0 {
		if e.stringStart("MC") && e.idx == 1 {
			//e.g., "McHugh"
			e.metaphAdd('K')
		} else {
			e.metaphAddAlt('X', 'K')
		}
	} else {
		e.metaphAdd('X')
	}

	e.idx++
	return true
}

func (e *Encoder) encodeChae() bool {
	// e.g. 'michael'
	if e.idx > 0 && e.stringAt(2, "AE") {
		if e.stringStart("RACHAEL") {
			e.metaphAdd('X')
		} else if !e.stringAt(-1, "C", "K", "G", "Q") {
			e.metaphAdd('K')
		}

		e.advanceCounter(3, 1)
		return true
	}

	return false
}

// Encodes transliterations from the hebrew where the
// sound 'kh' is represented as "-CH-". The normal pronounciation
// of this in english is either 'h' or 'kh', and alternate
// spellings most often use "-H-"
func (e *Encoder) encodeChToH() bool {
	// hebrew => 'H', e.g. 'channukah', 'chabad'
	if (e.idx == 0 &&
		(e.stringAt(2, "AIM", "ETH", "ELM", "ASID", "AZAN",
			"UPPAH", "UTZPA", "ALLAH", "ALUTZ", "AMETZ",
			"ESHVAN", "ADARIM", "ANUKAH", "ALLLOTH", "ANNUKAH", "AROSETH"))) ||
		e.stringAt(-3, "CLACHAN") {

		e.metaphAdd('H')
		e.advanceCounter(2, 1)
		return true
	}

	return false
}

func (e *Encoder) encodeSilentCh() bool {
	if e.stringAt(-2, "YACHT", "FUCHSIA") ||
		e.stringStart("STRACHAN", "CRICHTON") ||
		(e.stringAt(-3, "DRACHM") && !e.stringAt(-3, "DRACHMA")) {
		e.idx++
		return true
	}

	return false
}

func (e *Encoder) encodeChToX() bool {
	// e.g. 'approach', 'beach'
	if (e.stringAt(-2, "OACH", "EACH", "EECH", "OUCH", "OOCH", "MUCH", "SUCH") && !e.stringAt(-3, "JOACH")) ||
		e.stringAtEnd(-1, "ACHA", "ACHO") || // e.g. 'dacha', 'macho'
		e.stringAtEnd(0, "CHOT", "CHOD", "CHAT") ||
		(e.stringAtEnd(-1, "OCHE") && !e.stringAt(-2, "DOCHE")) ||
		e.stringAt(-4, "ATTACH", "DETACH", "KOVACH", "PARACHUT") ||
		e.stringAt(-5, "SPINACH", "MASSACHU") ||
		e.stringStart("MACHAU") ||
		(e.stringAt(-3, "THACH") && !e.stringAt(1, "E")) || // no ACHE
		e.stringAt(-2, "VACHON") {

		e.metaphAdd('X')
		e.idx++
		return true
	}

	return false
}

func (e *Encoder) encodeEnglishChToK() bool {
	//'ache', 'echo', alternate spelling of 'michael'
	if (e.idx == 1 && rootOrInflections(e.in, "ACHE")) ||
		((e.idx > 3 && rootOrInflections(e.in[e.idx-1:], "ACHE")) &&
			e.stringStart("EAR", "HEAD", "BACK", "HEART", "BELLY", "TOOTH")) ||
		e.stringAt(-1, "ECHO") ||
		e.stringAt(-2, "MICHAEL") ||
		e.stringAt(-4, "JERICHO") ||
		e.stringAt(-5, "LEPRECH") {

		e.metaphAddAlt('K', 'X')
		e.idx++
		return true
	}

	return false
}

func (e *Encoder) encodeGermanicChToK() bool {
	// various germanic
	// "<consonant><vowel>CH-"implies a german word where 'ch' => K

	if (e.idx > 1 &&
		!e.isVowelAt(-2) &&
		e.stringAt(-1, "ACH") &&
		!e.stringAt(-2, "MACHADO", "MACHUCA", "LACHANC", "LACHAPE", "KACHATU") &&
		!e.stringAt(-3, "KHACHAT") &&
		(!e.charAt(2, 'I') && (!e.charAt(2, 'E') || e.stringAt(-2, "BACHER", "MACHER", "MACHEN", "LACHER"))) ||
		// e.g. 'brecht', 'fuchs'
		(e.stringAt(2, "T", "S") && !(e.stringStart("WHICHSOEVER", "LUNCHTIME"))) ||
		// e.g. 'andromache'
		e.stringStart("SCHR") ||
		(e.idx > 2 && e.stringAt(-2, "MACHE")) ||
		(e.idx == 2 && e.stringAt(-2, "ZACH")) ||
		e.stringAt(-4, "SCHACH") ||
		e.stringAt(-1, "ACHEN") ||
		e.stringAt(-3, "SPICH", "ZURCH", "BUECH") ||
		(e.stringAt(-3, "KIRCH", "JOACH", "BLECH", "MALCH") && !(e.stringAt(-3, "KIRCHNER") || e.idx+1 == e.lastIdx)) || // "kirch" and "blech" both get 'X'
		e.stringAtEnd(-2, "NICH", "LICH", "BACH") ||
		e.stringAtEnd(-3, "URICH", "BRICH", "ERICH", "DRICH", "NRICH") ||
		e.stringAtEnd(-5, "ALDRICH") ||
		e.stringAtEnd(-6, "GOODRICH") ||
		e.stringAtEnd(-7, "GINGERICH")) ||
		e.stringAtEnd(-4, "ULRICH", "LFRICH", "LLRICH", "EMRICH", "ZURICH", "EYRICH") ||
		// e.g., 'wachtler', 'wechsler', but not 'tichner'
		((e.stringAt(-1, "A", "O", "U", "E") || e.idx == 0) &&
			e.stringAt(2, "L", "R", "N", "M", "B", "H", "F", "V", "W", " ")) {

		// "CHR/L-" e.g. 'chris' do not get
		// alt pronunciation of 'X'
		if e.stringAt(2, "R", "L") || e.isSlavoGermanic() {
			e.metaphAdd('K')
		} else {
			e.metaphAddAlt('K', 'X')
		}
		e.idx++
		return true
	}

	return false
}

// Encode "-ARCH-". Some occurances are from greek roots and therefore encode
// to 'K', others are from english words and therefore encode to 'X'
func (e *Encoder) encodeArch() bool {
	if e.stringAt(-2, "ARCH") {
		// "-ARCH-" has many combining forms where "-CH-" => K because of its
		// derivation from the greek
		if ((e.isVowelAt(2) && e.stringAt(-2, "ARCHA", "ARCHI", "ARCHO", "ARCHU", "ARCHY")) ||
			e.stringAt(-2, "ARCHEA", "ARCHEG", "ARCHEO", "ARCHET", "ARCHEL", "ARCHES", "ARCHEP", "ARCHEM", "ARCHEN") ||
			e.stringAtEnd(-2, "ARCH") ||
			e.stringStart("MENARCH")) &&
			(!rootOrInflections(e.in, "ARCH") &&
				!e.stringAt(-4, "SEARCH", "POARCH") &&
				!e.stringStart("ARCHER", "ARCHIE", "ARCHENEMY", "ARCHIBALD", "ARCHULETA", "ARCHAMBAU") &&
				!((((e.stringAt(-3, "LARCH", "MARCH", "PARCH") ||
					e.stringAt(-4, "STARCH")) &&
					!e.stringStart("EPARCH", "NOMARCH", "EXILARCH", "HIPPARCH", "MARCHESE", "ARISTARCH", "MARCHETTI")) ||
					rootOrInflections(e.in, "STARCH")) &&
					(!e.stringAt(-2, "ARCHU", "ARCHY") || e.stringStart("STARCHY")))) {

			e.metaphAddAlt('K', 'X')
		} else {
			e.metaphAdd('X')
		}
		e.idx++
		return true
	}

	return false
}

func (e *Encoder) encodeGreekChInitial() bool {
	// greek roots e.g. 'chemistry', 'chorus', ch at beginning of root
	if (e.stringAt(0, "CHAMOM", "CHARAC", "CHARIS", "CHARTO", "CHARTU", "CHARYB", "CHRIST", "CHEMIC", "CHILIA") ||
		(e.stringAt(0, "CHEMI", "CHEMO", "CHEMU", "CHEMY", "CHOND", "CHONA", "CHONI", "CHOIR", "CHASM",
			"CHARO", "CHROM", "CHROI", "CHAMA", "CHALC", "CHALD", "CHAET", "CHIRO", "CHILO", "CHELA", "CHOUS",
			"CHEIL", "CHEIR", "CHEIM", "CHITI", "CHEOP") && !(e.stringAt(0, "CHEMIN") || e.stringAt(-2, "ANCHONDO"))) ||
		(e.stringAt(0, "CHISM", "CHELI") &&
			// exclude spanish "machismo"
			!(e.stringStart("MICHEL", "MACHISMO", "RICHELIEU", "REVANCHISM") ||
				e.stringExact("CHISM"))) ||
		// include e.g. "chorus", "chyme", "chaos"
		(e.stringAt(0, "CHOR", "CHOL", "CHYM", "CHYL", "CHLO", "CHOS", "CHUS", "CHOE") && !e.stringStart("CHOLLO", "CHOLLA", "CHORIZ")) ||
		// "chaos" => K but not "chao"
		(e.stringAt(0, "CHAO") && e.idx+3 != e.lastIdx) ||
		// e.g. "abranchiate"
		(e.stringAt(0, "CHIA") && !(e.stringStart("CHIAPAS", "APPALACHIA"))) ||
		// e.g. "chimera"
		e.stringAt(0, "CHIMERA", "CHIMAER", "CHIMERI") ||
		// e.g. "chameleon"
		e.stringStart("CHAME", "CHELO", "CHITO") ||
		// e.g. "spirochete"
		((e.idx+4 == e.lastIdx || e.idx+5 == e.lastIdx) && e.stringAt(-1, "OCHETE"))) &&
		// more exceptions where "-CH-" => X e.g. "chortle", "crocheter"
		!(e.stringExact("CHORE", "CHOLO", "CHOLA") ||
			e.stringAt(0, "CHORT", "CHOSE") ||
			e.stringAt(-3, "CROCHET") ||
			e.stringStart("CHEMISE", "CHARISE", "CHARISS", "CHAROLE")) {

		if e.stringAt(2, "R", "L") {
			e.metaphAdd('K')
		} else {
			e.metaphAddAlt('K', 'X')
		}
		e.idx++
		return true
	}

	return false
}

func (e *Encoder) encodeGreekChNonInitial() bool {
	//greek & other roots e.g. 'tachometer', 'orchid', ch in middle or end of root
	if e.stringAt(-2, "LYCHN", "TACHO", "ORCHO", "ORCHI", "LICHO", "ORCHID", "NICHOL",
		"MECHAN", "LICHEN", "MACHIC", "PACHEL", "RACHIF", "RACHID",
		"RACHIS", "RACHIC", "MICHAL", "ORCHESTR") ||
		e.stringAt(-3, "MELCH", "GLOCH", "TRACH", "TROCH", "BRACH", "SYNCH", "PSYCH",
			"STICH", "PULCH", "EPOCH") ||
		(e.stringAt(-3, "TRICH") && !e.stringAt(-5, "OSTRICH")) ||
		(e.stringAt(-2, "TYCH", "TOCH", "BUCH", "MOCH", "CICH", "DICH", "NUCH", "EICH", "LOCH",
			"DOCH", "ZECH", "WYCH") && !(e.stringAt(-4, "INDOCHINA") || e.stringAt(-2, "BUCHON"))) ||
		((e.idx == 1 || e.idx == 2) && e.stringAt(-1, "OCHER", "ECHIN", "ECHID")) ||
		e.stringAt(-4, "BRONCH", "STOICH", "STRYCH", "TELECH", "PLANCH", "CATECH", "MANICH", "MALACH",
			"BIANCH", "DIDACH", "BRANCHIO", "BRANCHIF") ||
		e.stringStart("ICHA", "ICHN") ||
		(e.stringAt(-1, "ACHAB", "ACHAD", "ACHAN", "ACHAZ") && !e.stringAt(-2, "MACHADO", "LACHANC")) ||
		e.stringAt(-1, "ACHISH", "ACHILL", "ACHAIA", "ACHENE", "ACHAIAN", "ACHATES", "ACHIRAL", "ACHERON",
			"ACHILLEA", "ACHIMAAS", "ACHILARY", "ACHELOUS", "ACHENIAL", "ACHERNAR",
			"ACHALASIA", "ACHILLEAN", "ACHIMENES", "ACHIMELECH", "ACHITOPHEL") ||
		// e.g. 'inchoate'
		(e.idx == 2 && (e.stringStart("INCHOA")) ||
			// e.g. 'ischemia'
			e.stringStart("ISCH")) ||
		// e.g. 'ablimelech', 'antioch', 'pentateuch'
		(e.idx+1 == e.lastIdx && e.stringAt(-1, "A", "O", "U", "E") &&
			!(e.stringStart("DEBAUCH") || e.stringAt(-2, "MUCH", "SUCH", "KOCH") ||
				e.stringAt(-5, "OODRICH", "ALDRICH"))) {

		e.metaphAddAlt('K', 'X')
		e.idx++
		return true
	}

	return false
}

//Encodes reliably italian "-CCIA-"
func (e *Encoder) encodeCcia() bool {
	//e.g., 'focaccia'
	if e.stringAt(1, "CIA") {
		e.metaphAddAlt('X', 'S')
		e.idx++
		return true
	}

	return false
}

func (e *Encoder) encodeCc() bool {
	//double 'C', but not if e.g. 'McClellan'
	if e.stringAt(0, "CC") && !(e.idx == 1 && e.in[0] == 'M') {
		// exception
		if e.stringAt(-3, "FLACCID") {
			e.metaphAdd('S')
			e.advanceCounter(2, 1)
			return true
		}

		//'bacci', 'bertucci', other italian
		if e.stringAtEnd(2, "I") ||
			e.stringAt(2, "IO") || e.stringAtEnd(2, "INO", "INI") {
			e.metaphAdd('X')
			e.advanceCounter(2, 1)
			return true
		}

		//'accident', 'accede' 'succeed'
		if e.stringAt(2, "I", "E", "Y") && //except 'bellocchio','bacchus', 'soccer' get K
			!(e.charAt(2, 'H') || e.stringAt(-2, "SOCCER")) {
			e.metaphAddStr("KS", "KS")
			e.advanceCounter(2, 1)
			return true
		}
		// Pierce's rule
		e.metaphAdd('K')
		e.idx++
		return true
	}

	return false
}

func (e *Encoder) encodeCkCgCq() bool {
	if e.stringAt(0, "CK", "CG", "CQ") {

		// eastern european spelling e.g. 'gorecki' == 'goresky'
		if e.stringAtEnd(0, "CKI", "CKY") && len(e.in) > 6 {
			e.metaphAddStr("K", "SK")
		} else {
			e.metaphAdd('K')
		}
		if e.stringAt(2, "K", "G", "Q") {
			e.idx += 2
		} else {
			e.idx++
		}

		return true
	}

	return false
}

//Encode cases where "C" preceeds a front vowel such as "E", "I", or "Y".
//These cases most likely => S or X
func (e *Encoder) encodeCFrontVowel() bool {
	if e.stringAt(0, "CI", "CE", "CY") {
		if e.encodeBritishSilentCE() ||
			e.encodeCe() ||
			e.encodeCi() ||
			e.encodeLatinateSuffixes() {

			e.advanceCounter(1, 0)
			return true
		}

		e.metaphAdd('S')
		e.advanceCounter(1, 0)
		return true
	}

	return false
}

func (e *Encoder) encodeBritishSilentCE() bool {
	// english place names like e.g.'gloucester' pronounced glo-ster
	if e.stringAtEnd(1, "ESTER") || e.stringAt(1, "ESTERSHIRE") {
		return true
	}

	return false
}

func (e *Encoder) encodeCe() bool {
	// 'ocean', 'commercial', 'provincial', 'cello', 'fettucini', 'medici'
	if (e.stringAt(1, "EAN") && e.isVowelAt(-1)) ||
		(e.stringAtEnd(-1, "ACEA") && !e.stringStart("PANACEA")) || // e.g. 'rosacea'
		e.stringAt(1, "ELLI", "ERTO", "EORL") || // e.g. 'botticelli', 'concerto'
		e.stringAtEnd(-3, "CROCE") || // some italian names familiar to americans
		e.stringAt(-3, "DOLCE") ||
		e.stringAtEnd(1, "ELLO") { // e.g. cello

		e.metaphAddAlt('X', 'S')
		return true
	}

	return false
}

func (e *Encoder) encodeCi() bool {
	// with consonant before C
	// e.g. 'fettucini', but exception for the americanized pronunciation of 'mancini'

	if (e.stringAt(1, "INI") && !e.stringAtEnd(-e.idx, "MANCINI")) ||
		e.stringAtEnd(-1, "ICI") || // e.g. 'medici'
		e.stringAt(-1, "RCIAL", "NCIAL", "RCIAN", "UCIUS") || // e.g. "commercial', 'provincial', 'cistercian'
		e.stringAt(-3, "MARCIA") || // special cases
		e.stringAt(-2, "ANCIENT") {
		e.metaphAddAlt('X', 'S')
		return true
	}

	// exception
	if e.stringAt(-4, "COERCION") {
		e.metaphAdd('J')
		return true
	}

	// with vowel before C (or at beginning?)
	if (e.stringAt(0, "CIO", "CIE", "CIA") && e.isVowelAt(-1)) ||
		e.stringAt(1, "IAO") {

		if (e.stringAt(0, "CIAN", "CIAL", "CIAO", "CIES", "CIOL", "CION") ||
			e.stringAt(-3, "GLACIER") || // exception - "glacier" => 'X' but "spacier" = > 'S'
			e.stringAt(0, "CIENT", "CIENC", "CIOUS", "CIATE", "CIATI", "CIATO", "CIABL", "CIARY") ||
			e.stringAtEnd(0, "CIA", "CIO", "CIAS", "CIOS")) &&
			!(e.stringAt(-4, "ASSOCIATION") || e.stringStart("OCIE") ||
				// exceptions mostly because these names are usually from
				// the spanish rather than the italian in america
				e.stringAt(-2, "LUCIO", "SOCIO", "SOCIE", "MACIAS", "LUCIANO", "HACIENDA") ||
				e.stringAt(-3, "GRACIE", "GRACIA", "MARCIANO") ||
				e.stringAt(-4, "PALACIO", "POLICIES", "FELICIANO") ||
				e.stringAt(-5, "MAURICIO") ||
				e.stringAt(-6, "ANDALUCIA") ||
				e.stringAt(-7, "ENCARNACION")) {

			e.metaphAddAlt('X', 'S')
		} else {
			e.metaphAddAlt('S', 'X')
		}

		return true
	}

	return false
}

func (e *Encoder) encodeLatinateSuffixes() bool {
	if e.stringAt(1, "EOUS", "IOUS") {
		e.metaphAddAlt('X', 'S')
		return true
	}
	return false
}

func (e *Encoder) encodeSilentC() bool {
	if e.stringAt(1, "T", "S") && e.stringStart("INDICT", "TUCSON", "CONNECTICUT") {
		return true
	}

	return false
}

// Encodes slavic spellings or transliterations
// written as "-CZ-"
func (e *Encoder) encodeCz() bool {
	if e.stringAt(1, "Z") && !e.stringAt(-1, "ECZEMA") {
		if e.stringAt(0, "CZAR") {
			e.metaphAdd('S')
		} else {
			// otherwise most likely a czech word...
			e.metaphAdd('X')
		}
		e.idx++
		return true
	}

	return false
}

func (e *Encoder) encodeCs() bool {
	// give an 'etymological' 2nd
	// encoding for "kovacs" so
	// that it matches "kovach"

	if e.stringStart("KOVACS") {
		e.metaphAddStr("KS", "X")
		e.idx++
		return true
	}

	if e.stringAt(-1, "ACS") && !e.stringAtEnd(-4, "ISAACS") {
		e.metaphAdd('X')
		e.idx++
		return true
	}

	return false
}

func (e *Encoder) encodeD() { panic("not implemented") }
func (e *Encoder) encodeF() { panic("not implemented") }
func (e *Encoder) encodeG() { panic("not implemented") }
func (e *Encoder) encodeH() { panic("not implemented") }

func (e *Encoder) encodeJ() {
	if e.encodeSpanishJ() || e.encodeSpanishOjUj() {
		return
	}

	e.encodeOtherJ()
}

func (e *Encoder) encodeSpanishJ() bool {
	//obvious spanish, e.g. "jose", "san jacinto"
	if (e.stringAt(1, "UAN", "ACI", "ALI", "EFE", "ICA", "IME", "OAQ", "UAR") &&
		!e.stringAt(0, "JIMERSON", "JIMERSEN")) ||
		e.stringAtEnd(1, "OSE") ||
		e.stringAt(1, "EREZ", "UNTA", "AIME", "AVIE", "AVIA", "IMINEZ", "ARAMIL") ||
		e.stringAtEnd(-2, "MEJIA") ||
		e.stringAt(-2, "TEJED", "TEJAD", "LUJAN", "FAJAR", "BEJAR", "BOJOR", "CAJIG",
			"DEJAS", "DUJAR", "DUJAN", "MIJAR", "MEJOR", "NAJAR",
			"NOJOS", "RAJED", "RIJAL", "REJON", "TEJAN", "UIJAN") ||
		e.stringAt(-3, "ALEJANDR", "GUAJARDO", "TRUJILLO") ||
		(e.stringAt(-2, "RAJAS") && e.idx > 2) ||
		(e.stringAt(-2, "MEJIA") && !e.stringAt(-2, "MEJIAN")) ||
		e.stringAt(-1, "OJEDA") ||
		e.stringAt(-3, "LEIJA", "MINJA", "VIAJES", "GRAJAL") ||
		e.stringAt(0, "JAUREGUI") ||
		e.stringAt(-4, "HINOJOSA") ||
		e.stringStart("SAN ") ||
		((e.idx+1 == e.lastIdx) && e.charAt(1, 'O') && !e.stringStart("TOJO", "BANJO", "MARYJO")) {

		// americans pronounce "juan" as 'wan'
		// and "marijuana" and "tijuana" also
		// do not get the 'H' as in spanish, so
		// just treat it like a vowel in these cases

		if !(e.stringAt(0, "JUAN") || e.stringAt(0, "JOAQ")) {
			e.metaphAdd('H')
		} else if e.idx == 0 {
			e.metaphAdd('A')
		}
		e.advanceCounter(1, 0)
		return true
	}

	// Jorge gets 2nd HARHA. also JULIO, JESUS
	if e.stringAt(1, "ORGE", "ULIO", "ESUS") && !e.stringStart("JORGEN") {
		// get both consonants for "jorge"
		if e.stringAtEnd(1, "ORGE") {
			if e.EncodeVowels {
				e.metaphAddStr("JARJ", "HARHA")
			} else {
				e.metaphAddStr("JRJ", "HRH")
			}
			e.advanceCounter(4, 4)
			return true
		}
		e.metaphAddAlt('J', 'H')
		e.advanceCounter(1, 0)
		return true
	}

	return false
}

func (e *Encoder) encodeSpanishOjUj() bool {
	if e.stringAt(1, "OJOBA", "UJUY") {
		if e.EncodeVowels {
			e.metaphAddStr("HAH", "HAH")
		} else {
			e.metaphAddStr("HH", "HH")
		}

		e.advanceCounter(3, 2)
		return true
	}

	return false
}

func (e *Encoder) encodeOtherJ() { panic("not implemented") }

func (e *Encoder) encodeK() {
	if !e.encodeSilentK() {
		e.metaphAdd('K')

		// eat redundant K's and Q's
		if e.charAt(1, 'K') || e.charAt(1, 'Q') {
			e.idx++
		}
	}
}

func (e *Encoder) encodeSilentK() bool {
	if e.idx == 0 && e.stringStart("KN") {
		if !e.stringAt(2, "ISH", "ESSET", "IEVEL") {
			return true
		}
	}

	// e.g. "know", "knit", "knob"
	if (e.stringAt(1, "NOW", "NIT", "NOT", "NOB") && !e.stringStart("BANKNOTE")) ||
		e.stringAt(1, "NOCK", "NUCK", "NIFE", "NACK", "NIGHT") {
		// N already encoded before
		// e.g. "penknife"
		if e.idx > 0 && e.charAt(-1, 'N') {
			e.idx++
		}

		return true
	}

	return false
}

func (e *Encoder) encodeL() { panic("not implemented") }
func (e *Encoder) encodeM() { panic("not implemented") }
func (e *Encoder) encodeN() { panic("not implemented") }
func (e *Encoder) encodeP() { panic("not implemented") }
func (e *Encoder) encodeQ() { panic("not implemented") }
func (e *Encoder) encodeR() { panic("not implemented") }
func (e *Encoder) encodeS() { panic("not implemented") }
func (e *Encoder) encodeT() { panic("not implemented") }
func (e *Encoder) encodeV() { panic("not implemented") }
func (e *Encoder) encodeW() { panic("not implemented") }
func (e *Encoder) encodeX() { panic("not implemented") }
func (e *Encoder) encodeZ() { panic("not implemented") }

func (e *Encoder) encodeVowels() {

	if e.idx == 0 {
		// all init vowels map to 'A'
		// as of Double Metaphone
		e.metaphAdd('A')
	} else if e.EncodeVowels {
		if !e.charAt(e.idx, 'E') {
			if e.encodeSkipSilentUe() {
				return
			}
			if e.encodeOSilent() {
				return
			}
			// encode all vowels and
			// diphthongs to the same value
			e.metaphAdd('A')

		} else {
			e.encodeEPronounced()
		}
	}

	if !(!e.isVowelAt(-2) && e.stringAt(-1, "LEWA", "LEWO", "LEWI")) {
		e.idx = e.skipVowels(e.idx)
	}
}

func (e *Encoder) encodeSkipSilentUe() bool {
	// always silent except for cases listed below
	if (e.stringAt(-1, "QUE", "GUE") &&
		!e.stringStart("RISQUE", "PIROGUE", "ENRIQUE", "BARBEQUE", "PALENQUE", "APPLIQUE", "COMMUNIQUE") &&
		!e.stringAt(-3, "ARGUE", "SEGUE")) &&
		e.idx > 1 &&
		((e.idx+1 == e.lastIdx) || e.stringStart("JACQUES")) {

		e.idx = e.skipVowels(e.idx)
		return true
	}
	return false
}

// Encodes cases where non-initial 'e' is pronounced, taking
// care to detect unusual cases from the greek.
// Only executed if non initial vowel encoding is turned on
func (e *Encoder) encodeEPronounced() {
	// special cases with two pronunciations
	// 'agape' 'lame' 'resume'
	if e.stringExact("LAME", "SAKE", "PATE") ||
		e.stringExact("AGAPE") ||
		(e.stringStart("RESUME") && e.idx == 5) {

		e.metaphAddAlt(unicode.ReplacementChar, 'A')
		return
	}

	// special case "inge" => 'INGA', 'INJ'
	if e.stringExact("INGE") {
		e.metaphAddAlt('A', unicode.ReplacementChar)
		return
	}

	// special cases with two pronunciations
	// special handling due to the difference in
	// the pronunciation of the '-D'
	if e.idx == 5 && e.stringStart("BLESSED", "LEARNED") {
		e.metaphAddExactApproxAlt("D", "AD", "T", "AT")
		e.idx++
		return
	}

	// encode all vowels and diphthongs to the same value
	if (!e.encodeESilent() && !e.flagAlInversion && !e.encodeSilentInternalE()) ||
		e.encodeEPronouncedExceptions() {

		e.metaphAdd('A')
	}

	// now that we've visited the vowel in question
	e.flagAlInversion = false
}

func (e *Encoder) encodeOSilent() bool {
	// if "iron" at beginning or end of word and not "irony"
	if e.charAt(0, 'O') {
		if (e.stringStart("IRON") || e.stringAtEnd(-2, "IRON")) && !e.stringAt(-2, "IRONIC") {
			return true
		}
	}

	return false
}

func (e *Encoder) encodeESilent() bool {
	if e.encodeEPronouncedAtEnd() {
		return false
	}

	// 'e' silent when last letter, altho
	if e.idx == e.lastIdx ||
		// also silent if before plural 's'
		// or past tense or participle 'd', e.g.
		// 'grapes' and 'banished' => PNXT
		(e.idx > 1 && e.idx+1 == e.lastIdx && e.stringAt(1, "S", "D") &&
			// and not e.g. "nested", "rises", or "pieces" => RASAS
			!(e.stringAt(-1, "TED", "SES", "CES") ||
				e.stringStart("ABED", "IMED", "JARED", "AHMED", "HAMED", "JAVED",
					"NORRED", "MEDVED", "MERCED", "ALLRED", "KHALED", "RASHED", "MASJED",
					"MOHAMED", "MOHAMMED", "MUHAMMED", "MOUHAMED", "ANTIPODES", "ANOPHELES"))) ||
		// e.g.  'wholeness', 'boneless', 'barely'
		e.stringAtEnd(1, "NESS", "LESS") ||
		(e.stringAtEnd(1, "LY") && !e.stringStart("CICELY")) {

		return true
	}
	return false
}

// Tests for words where an 'E' at the end of the word
// is pronounced
//
// special cases, mostly from the greek, spanish, japanese,
// italian, and french words normally having an acute accent.
// also, pronouns and articles
//
// Many Thanks to ali, QuentinCompson, JeffCO, ToonScribe, Xan,
// Trafalz, and VictorLaszlo, all of them atriots from the Eschaton,
// for all their fine contributions!
func (e *Encoder) encodeEPronouncedAtEnd() bool {
	if e.idx == e.lastIdx &&
		(e.stringAt(-6, "STROPHE") ||
			// if a vowel is before the 'E', vowel eater will have eaten it.
			//otherwise, consonant + 'E' will need 'E' pronounced
			len(e.in) == 2 ||
			(len(e.in) == 3 && !e.isVowelAt(0)) ||
			// these german name endings can be relied on to have the 'e' pronounced
			(e.stringAtEnd(-2, "BKE", "DKE", "FKE", "KKE", "LKE", "NKE", "MKE", "PKE", "TKE", "VKE", "ZKE") &&
				!e.stringStart("FINKE", "FUNKE", "FRANKE")) ||
			e.stringAtEnd(-4, "SCHKE") ||
			e.stringExact("ACME", "NIKE", "CAFE", "RENE", "LUPE", "JOSE", "ESME",
				"LETHE", "CADRE", "TILDE", "SIGNE", "POSSE", "LATTE", "ANIME", "DOLCE", "CROCE",
				"ADOBE", "OUTRE", "JESSE", "JAIME", "JAFFE", "BENGE", "RUNGE",
				"CHILE", "DESME", "CONDE", "URIBE", "LIBRE", "ANDRE",
				"HECATE", "PSYCHE", "DAPHNE", "PENSKE", "CLICHE", "RECIPE",
				"TAMALE", "SESAME", "SIMILE", "FINALE", "KARATE", "RENATE", "SHANTE",
				"OBERLE", "COYOTE", "KRESGE", "STONGE", "STANGE", "SWAYZE", "FUENTE",
				"SALOME", "URRIBE",
				"ECHIDNE", "ARIADNE", "MEINEKE", "PORSCHE", "ANEMONE", "EPITOME",
				"SYNCOPE", "SOUFFLE", "ATTACHE", "MACHETE", "KARAOKE", "BUKKAKE",
				"VICENTE", "ELLERBE", "VERSACE",
				"PENELOPE", "CALLIOPE", "CHIPOTLE", "ANTIGONE", "KAMIKAZE", "EURIDICE",
				"YOSEMITE", "FERRANTE",
				"HYPERBOLE", "GUACAMOLE", "XANTHIPPE",
				"SYNECDOCHE")) {

		return true
	}

	return false
}

func (e *Encoder) encodeSilentInternalE() bool {
	// 'olesen' but not 'olen'	RAKE BLAKE
	if (e.stringStart("OLE") && e.encodeESuffix(3)) ||
		(e.stringStart("BARE", "FIRE", "FORE", "GATE", "HAGE", "HAVE",
			"HAZE", "HOLE", "CAPE", "HUSE", "LACE", "LINE",
			"LIVE", "LOVE", "MORE", "MOSE", "MORE", "NICE",
			"RAKE", "ROBE", "ROSE", "SISE", "SIZE", "WARE",
			"WAKE", "WISE", "WINE") && e.encodeESuffix(4)) ||
		(e.stringStart("BLAKE", "BRAKE", "BRINE", "CARLE", "CLEVE", "DUNNE",
			"HEDGE", "HOUSE", "JEFFE", "LUNCE", "STOKE", "STONE",
			"THORE", "WEDGE", "WHITE") && e.encodeESuffix(5)) ||
		(e.stringStart("BRIDGE", "CHEESE") && e.encodeESuffix(6)) ||
		(e.stringAt(-5, "CHARLES")) {
		return true
	}

	return false
}

func (e *Encoder) encodeESuffix(at int) bool {
	//E_Silent_Suffix && !E_Pronouncing_Suffix

	if e.idx == at-1 && len(e.in) > at+1 &&
		(e.isVowelAt(-e.idx+at+1) ||
			(e.stringAt(-e.idx+at, "ST", "SL") && len(e.in) > at+2)) {

		// now filter endings that will cause the 'e' to be pronounced

		// e.g. 'bridgewood' - the other vowels will get eaten
		// up so we need to put one in here
		// e.g. 'bridgette'
		// e.g. 'olena'
		// e.g. 'bridget'
		if e.stringAtEnd(-e.idx+at, "T", "R", "TA", "TT", "NA", "NO", "NE",
			"RS", "RE", "LA", "AU", "RO", "RA", "TTE", "LIA", "NOW", "ROS", "RAS",
			"WOOD", "WATER", "WORTH") {
			return false
		}

		return true
	}

	return false
}

// Exceptions where 'E' is pronounced where it
// usually wouldn't be, and also some cases
// where 'LE' transposition rules don't apply
// and the vowel needs to be encoded here
func (e *Encoder) encodeEPronouncedExceptions() bool {
	// greek names e.g. "herakles" or hispanic names e.g. "robles", where 'e' is pronounced, other exceptions
	if (e.idx+1 == e.lastIdx &&
		(e.stringAtEnd(-3, "OCLES", "ACLES", "AKLES") ||
			e.stringStart("INES",
				"LOPES", "ESTES", "GOMES", "NUNES", "ALVES", "ICKES",
				"INNES", "PERES", "WAGES", "NEVES", "BENES", "DONES",
				"CORTES", "CHAVES", "VALDES", "ROBLES", "TORRES", "FLORES", "BORGES",
				"NIEVES", "MONTES", "SOARES", "VALLES", "GEDDES", "ANDRES", "VIAJES",
				"CALLES", "FONTES", "HERMES", "ACEVES", "BATRES", "MATHES",
				"DELORES", "MORALES", "DOLORES", "ANGELES", "ROSALES", "MIRELES", "LINARES",
				"PERALES", "PAREDES", "BRIONES", "SANCHES", "CAZARES", "REVELES", "ESTEVES",
				"ALVARES", "MATTHES", "SOLARES", "CASARES", "CACERES", "STURGES", "RAMIRES",
				"FUNCHES", "BENITES", "FUENTES", "PUENTES", "TABARES", "HENTGES", "VALORES",
				"GONZALES", "MERCEDES", "FAGUNDES", "JOHANNES", "GONSALES", "BERMUDES",
				"CESPEDES", "BETANCES", "TERRONES", "DIOGENES", "CORRALES", "CABRALES",
				"MARTINES", "GRAJALES",
				"CERVANTES", "FERNANDES", "GONCALVES", "BENEVIDES", "CIFUENTES", "SIFUENTES",
				"SERVANTES", "HERNANDES", "BENAVIDES",
				"ARCHIMEDES", "CARRIZALES", "MAGALLANES"))) ||
		e.stringAt(-2, "FRED", "DGES", "DRED", "GNES") ||
		e.stringAt(-5, "PROBLEM", "RESPLEN") ||
		e.stringAt(-4, "REPLEN") ||
		e.stringAt(-3, "SPLE") {

		return true
	}

	return false
}

//////////////////////////////////////////////////////////////////////////////////////////////////////
// Functions to identify patterns
//////////////////////////////////////////////////////////////////////////////////////////////////////

// isVowel returns true for vowels in many languages and charactersets.
func isVowel(inChar rune) bool {
	return (inChar == 'A') || (inChar == 'E') || (inChar == 'I') || (inChar == 'O') || (inChar == 'U') || (inChar == 'Y') ||
		(inChar == 'À') || (inChar == 'Á') || (inChar == 'Â') || (inChar == 'Ã') || (inChar == 'Ä') || (inChar == 'Å') || (inChar == 'Æ') ||
		(inChar == 'È') || (inChar == 'É') || (inChar == 'Ê') || (inChar == 'Ë') ||
		(inChar == 'Ì') || (inChar == 'Í') || (inChar == 'Î') || (inChar == 'Ï') ||
		(inChar == 'Ò') || (inChar == 'Ó') || (inChar == 'Ô') || (inChar == 'Õ') || (inChar == 'Ö') || (inChar == 'Ø') ||
		(inChar == 'Ù') || (inChar == 'Ú') || (inChar == 'Û') || (inChar == 'Ü') || (inChar == 'Ý') ||
		(inChar == '\uC29F') || (inChar == '\uC28C')
}

/**
 * Tests whether the word is the root or a regular english inflection
 * of it, e.g. "ache", "achy", "aches", "ached", "aching", "achingly"
 * This is for cases where we want to match only the root and corresponding
 * inflected forms, and not completely different words which may have the
 * same substring in them.
 */
func rootOrInflections(inWord []rune, root string) bool {
	lenDiff := len(inWord) - len(root)

	// there's no alternate shorter than the root itself
	if lenDiff < 0 {
		return false
	}

	// inWord must start with all the letters of root except the last
	last := len(root) - 1
	for i := 0; i < last; i++ {
		if inWord[i] != rune(root[i]) {
			return false
		}
	}

	inWord = inWord[last:]
	// at this point we know they start the same way
	// except the last rune of root that we didn't check, so check that now
	// check our last letter and simple plural

	if inWord[0] == rune(root[last]) {
		// last root letter matches
		if lenDiff == 0 {
			//exact match
			return true
		} else if lenDiff == 1 && inWord[1] == 'S' {
			// match with an extra S
			return true
		}
	}

	// different paths if the last letter is 'E' or not
	if root[last] == 'E' {
		// check ED
		if lenDiff == 1 && inWord[0] == 'E' && inWord[1] == 'D' {
			return true
		}
	} else {
		// check +ES
		// check +ED
		// the last character must match if the root doesn't end in E
		if inWord[0] != rune(root[last]) {
			return false
		}

		if lenDiff == 2 &&
			inWord[1] == 'E' && (inWord[2] == 'S' || inWord[2] == 'D') {
			return true
		}
	}

	// at this point our root and inWord match, so now we're just checking the endings
	// of the inWord starting at index "last"

	if lenDiff == 3 && areEqual(inWord, []rune("ING")) {
		// check ING
		return true
	} else if lenDiff == 5 && areEqual(inWord, []rune("INGLY")) {
		// check INGLY
		return true
	} else if lenDiff == 1 && inWord[0] == 'Y' {
		// check Y
		return true
	}

	return false
}

func (e *Encoder) isSlavoGermanic() bool {
	return e.stringStart("SCH", "SW") || e.in[0] == 'J' || e.in[0] == 'W'
}

func (e *Encoder) charNextIs(c rune) bool {
	return e.charAt(1, c)
}

func (e *Encoder) isVowelAt(offset int) bool {
	at := e.idx + offset
	if at < 0 || at >= len(e.in) {
		return false
	}

	return isVowel(e.in[at])
}

func (e *Encoder) charAt(offset int, c rune) bool {
	idx := e.idx + offset
	if idx >= len(e.in) {
		return false
	}

	return e.in[idx] == c
}

// stringAt returns true if one of the given substrings is located at the
// relative offset (relative to current idx) given and uses all the remaining
// letters of the input.  The vals must be given in order of length, shortest to longest, all caps.
func (e *Encoder) stringAtEnd(offset int, vals ...string) bool {
	start := e.idx + offset

	// basic bounds check on our start plus
	// if our shortest input would make us run out of chars then none of our inputs could match
	if start < 0 || start >= len(e.in) || start+len(vals[0]) > len(e.in) {
		return false
	}

	// each value given
nextVal:
	for _, v := range vals {
		// bounds check - if we overrun then we know the rest of the list is too long
		// so we're done
		if last, inlen := start+len(v), len(e.in); last > inlen {
			return false
		} else if last < inlen {
			// if we don't land on exactly the end of the input string then we don't need
			// to check the letters
			continue nextVal
		}

		// each letter of the value given
		i := 0
		for _, c := range v {
			if c != e.in[start+i] {
				// char mis-match, this word is done
				continue nextVal
			}
			i++
		}

		// if we make it here we matched all letters
		return true
	}

	// if we make it here we've tried all vals and failed
	return false
}

// stringAt returns true if one of the given substrings is located at the
// relative offset (relative to current idx) given.  The vals must be given in order of
// length, shortest to longest, all caps.
func (e *Encoder) stringAt(offset int, vals ...string) bool {
	start := e.idx + offset

	// basic bounds check on our start plus
	// if our shortest input would make us run out of chars then none of our inputs could match
	if start < 0 || start >= len(e.in) || start+len(vals[0]) > len(e.in) {
		return false
	}

	// each value given
nextVal:
	for _, v := range vals {
		// bounds check - if we fail then we know the rest of the list is too long
		// so we're done
		if start+len(v) > len(e.in) {
			return false
		}

		// each letter of the value given
		i := 0
		for _, c := range v {
			if c != e.in[start+i] {
				// char mis-match, this word is done
				continue nextVal
			}
			i++
		}

		// if we make it here we matched all letters
		return true
	}

	// if we make it here we've tried all vals and failed
	return false
}

func (e *Encoder) stringStart(vals ...string) bool {
	return e.stringAt(-e.idx, vals...)
}

func (e *Encoder) stringExact(vals ...string) bool {
	// each value given
nextVal:
	for _, v := range vals {
		// bounds check - if we fail then we know the rest of the list is too long
		// so we're done
		if len(v) > len(e.in) {
			return false
		} else if len(v) < len(e.in) {
			// too short, next option
			continue nextVal
		}

		i := 0
		for _, c := range v {
			if c != e.in[i] {
				// char mis-match, this word is done
				continue nextVal
			}
			i++
		}

		// if we make it here we matched all letters
		return true
	}

	// if we make it here we've tried all vals and failed
	return false
}

//////////////////////////////////////////////////////////////////////////////////////////////////////
// Functions to mutate the outputs
//////////////////////////////////////////////////////////////////////////////////////////////////////

// Adds encoding character to the encoded string (primary and secondary)
func (e *Encoder) metaphAdd(in rune) {
	e.metaphAddAlt(in, in)
}

// Adds given encoding characters to the associated encoded strings
func (e *Encoder) metaphAddAlt(prim, second rune) {
	if prim != unicode.ReplacementChar {
		// don't dupe added A's
		if !(prim == 'A' && len(e.primBuf) > 0 && e.primBuf[len(e.primBuf)-1] == 'A') {
			e.primBuf = append(e.primBuf, prim)
		}
	}

	if second != unicode.ReplacementChar {
		// don't dupe added A's
		if !(second == 'A' && len(e.secondBuf) > 0 && e.secondBuf[len(e.secondBuf)-1] == 'A') {
			e.secondBuf = append(e.secondBuf, second)
		}
	}
}

// Adds given strings to the associated encoded strings
func (e *Encoder) metaphAddStr(prim, second string) {
	// don't dupe added A's
	if !(prim == "A" && len(e.primBuf) > 0 && e.primBuf[len(e.primBuf)-1] == 'A') {
		e.primBuf = append(e.primBuf, []rune(prim)...)
	}

	// don't dupe added A's
	if !(second == "A" && len(e.secondBuf) > 0 && e.secondBuf[len(e.secondBuf)-1] == 'A') {
		e.secondBuf = append(e.secondBuf, []rune(second)...)
	}
}

func (e *Encoder) metaphAddExactApproxAlt(exact, altExact, main, alt string) {
	if e.EncodeExact {
		e.metaphAddStr(exact, altExact)
	} else {
		e.metaphAddStr(main, alt)
	}
}

func (e *Encoder) metaphAddExactApprox(exact, main string) {
	if e.EncodeExact {
		e.metaphAddStr(exact, exact)
	} else {
		e.metaphAddStr(main, main)
	}
}

func (e *Encoder) skipVowels(at int) int {
	if at < 0 {
		return 0
	}
	if at >= len(e.in) {
		return len(e.in)
	}

	it := e.in[at]
	off := e.idx - at

	for isVowel(it) || it == 'W' {

		if e.stringAt(off, "WICZ", "WITZ", "WIAK") ||
			e.stringAt(off-1, "EWSKI", "EWSKY", "OWSKI", "OWSKY") ||
			e.stringAtEnd(off, "WICKI", "WACKI") {
			break
		}

		off++
		if e.charAt(off-1, 'W') &&
			e.charAt(off, 'H') &&
			!e.stringAt(off, "HOP", "HIDE", "HARD", "HEAD", "HAWK", "HERD", "HOOK", "HAND", "HOLE",
				"HEART", "HOUSE", "HOUND", "HAMMER") {

			off++
		}

		if at+off > e.lastIdx {
			break
		}

		it = e.in[at+off]
	}

	return at + off - 1
}

func (e *Encoder) advanceCounter(noEncodeVowel, encodeVowel int) {
	if e.EncodeVowels {
		e.idx += encodeVowel
	} else {
		e.idx += noEncodeVowel
	}
}

//////////////////////////////////////////////////////////////////////////////////////////////////////
// Misc helper functions
//////////////////////////////////////////////////////////////////////////////////////////////////////

func areEqual(buf1 []rune, buf2 []rune) bool {
	if len(buf1) != len(buf2) {
		return false
	}

	for i := 0; i < len(buf1); i++ {
		if buf1[i] != buf2[i] {
			return false
		}
	}

	return true
}

// make sure we have capacity for our whole buffer, but 0 len
func primeBuf(buf []rune, ensureCap int) []rune {
	if want := ensureCap - cap(buf); want > 0 {
		buf = make([]rune, 0, ensureCap)
	} else if len(buf) != 0 {
		buf = buf[0:0]
	}

	return buf
}