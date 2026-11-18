package obelisk

import (
	"crypto/rand"
	"encoding/binary"
)

// New instance ids are stable route slugs (adjective-animal) used in edge hostnames
// and Kong paths. DisplayName can change without affecting URLs.
var routeSlugAnimals = []string{
	"otter", "hawk", "lynx", "panda", "finch", "badger", "beaver", "bison",
	"camel", "crane", "dingo", "eagle", "egret", "falcon", "ferret", "gecko",
	"heron", "ibis", "jaguar", "koala", "lemur", "mink", "moose", "narwhal",
	"okapi", "orca", "quail", "raven", "salmon", "serval", "sloth", "stork",
	"tiger", "viper", "walrus", "yak", "zebra", "condor", "coyote", "dolphin",
	"alpaca", "ant", "bear", "bee", "bird", "boar", "bull", "bunny", "cat",
	"cheetah", "cobra", "cod", "collie", "cougar", "crow", "deer", "dog",
	"dove", "duck", "elk", "emu", "fish", "fox", "frog", "gaur", "gnat",
	"goat", "goose", "gopher", "gull", "hare", "horse", "hyena", "iguana",
	"impala", "jay", "kite", "kiwi", "lark", "lion", "llama", "locust",
	"macaw", "magpie", "manatee", "marten", "mole", "mouse", "newt", "owl",
	"ox", "parrot", "pelican", "pig", "pigeon", "pony", "puma", "rat",
	"robin", "seal", "shark", "sheep", "shrew", "skunk", "snail", "snake",
	"sparrow", "squid", "squirrel", "swan", "tern", "toad", "trout", "tuna",
	"turtle", "vole", "wasp", "whale", "wolf", "worm", "wren",
}

var routeSlugAdjectives = []string{
	"calm", "swift", "bold", "quiet", "amber", "brisk", "clever", "daring",
	"eager", "gentle", "hazy", "icy", "jolly", "keen", "lively", "mellow",
	"nimble", "opal", "plaid", "quick", "rustic", "sunny", "tidy", "vivid",
	"warm", "young", "zesty", "crisp", "fresh", "grand", "happy", "steady",
	"active", "agile", "alert", "aloof", "ample", "ancient", "ardent", "astute",
	"awake", "azure", "blithe", "brazen", "bright", "bronze", "bubbly", "busy",
	"candid", "chic", "civic", "civil", "clear", "close", "coral", "cosy",
	"cozy", "crazy", "cream", "curly", "cute", "dapper", "dark", "dawn",
	"deep", "deft", "dense", "dim", "direct", "dizzy", "drab", "dry",
	"duke", "early", "east", "easy", "elite", "empty", "epic", "even",
	"fair", "fancy", "fast", "feral", "fierce", "fine", "flat", "fleet",
	"fluid", "focal", "frank", "free", "full", "funky", "fuzzy", "glad",
	"glum", "gold", "good", "gray", "great", "green", "grim", "gross",
	"hard", "harsh", "heady", "heavy", "high", "holy", "honest", "hot",
	"huge", "humid", "ideal", "idle", "inner", "iron", "jade", "jazz",
	"joint", "just", "kind", "late", "lean", "light", "lime", "lithe",
	"local", "lofty", "lone", "long", "loose", "loud", "loyal", "lunar",
	"lush", "magic", "major", "maple", "merry", "mild", "mint", "misty",
	"modal", "moody", "moral", "mossy", "mute", "naive", "naval", "neat",
	"new", "noble", "north", "novel", "odd", "olive", "open", "pale",
	"peach", "pearl", "plain", "plump", "polar", "proud", "pure", "rapid",
	"rare", "raw", "real", "regal", "rich", "ripe", "rocky", "round",
	"royal", "ruddy", "safe", "sage", "salt", "sand", "sandy", "sharp",
	"sheer", "shiny", "short", "shy", "silky", "silver", "simple", "sleek",
	"slim", "slow", "small", "smart", "smoky", "snug", "soft", "solid",
	"sonic", "south", "spare", "sparse", "speck", "spicy", "still", "stout",
	"stray", "strict", "strong", "subtle", "sudden", "super", "sure", "sweet",
	"tall", "tame", "tan", "teal", "tense", "thick", "thin", "tight",
	"tiny", "tired", "top", "tough", "trim", "true", "twin", "ultra",
	"urban", "urgent", "vague", "vain", "valid", "valiant", "vanilla", "vast",
	"veiled", "velvet", "vexed", "vital", "vocal", "wild", "wide", "wise",
	"wiry", "witty", "woven", "zany", "zero", "zinc", "zippy",
}

func randomRouteSlug() string {
	animal := routeSlugAnimals[randomWordIndex(len(routeSlugAnimals))]
	adj := routeSlugAdjectives[randomWordIndex(len(routeSlugAdjectives))]
	return adj + "-" + animal
}

func randomWordIndex(n int) int {
	if n <= 0 {
		return 0
	}
	var b [4]byte
	_, _ = rand.Read(b[:])
	return int(binary.BigEndian.Uint32(b[:]) % uint32(n))
}