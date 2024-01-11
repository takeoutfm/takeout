// Copyright 2024 defsub
//
// This file is part of Takeout.
//
// Takeout is free software: you can redistribute it and/or modify it under the
// terms of the GNU Affero General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option)
// any later version.
//
// Takeout is distributed in the hope that it will be useful, but WITHOUT ANY
// WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
// FOR A PARTICULAR PURPOSE.  See the GNU Affero General Public License for
// more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with Takeout.  If not, see <https://www.gnu.org/licenses/>.

// Package config collects all configuration for the server with a single model
// which allows for easy viper-based configuration files.
package config

// auto-generated
var somafmStreams = []RadioStream{
	{
		Creator:     "SomaFM",
		Title:       "Beat Blender",
		Description: "A late night blend of deep-house and downtempo chill.",
		Image:       "https://somafm.com/img3/beatblender-400.png",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/beatblender130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/beatblender.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Black Rock FM",
		Description: "From the Playa to the world, for the annual Burning Man festival.",
		Image:       "https://somafm.com/img3/brfm-400.png",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/brfm130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/brfm.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Boot Liquor",
		Description: "Americana Roots music for Cowhands, Cowpokes and Cowtippers",
		Image:       "https://somafm.com/img3/bootliquor-400.png",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/bootliquor130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/bootliquor320.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Bossa Beyond",
		Description: "Silky-smooth, laid-back Brazilian-style rhythms of Bossa Nova, Samba and beyond",
		Image:       "https://somafm.com/logos/400/bossa-400.jpg",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/bossa130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/bossa256.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Christmas Lounge",
		Description: "Chilled holiday grooves and classic winter lounge tracks. (Kid and Parent safe!)",
		Image:       "https://somafm.com/img3/christmas-400.jpg",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/christmas130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/christmas256.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Christmas Rocks!",
		Description: "Have your self an indie/alternative holiday season!",
		Image:       "https://somafm.com/img3/xmasrocks-400.png",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/xmasrocks130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/xmasrocks.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "cliqhop idm",
		Description: "Blips'n'beeps backed mostly w/beats. Intelligent Dance Music.",
		Image:       "https://somafm.com/img3/xmasrocks-400.png",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/cliqhop130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/cliqhop256.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Covers",
		Description: "Just covers. Songs you know by artists you don't. We've got you covered.",
		Image:       "https://somafm.com/img3/covers-400.png",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/covers130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/covers.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Deep Space One",
		Description: "Deep ambient electronic, experimental and space music. For inner and outer space exploration.",
		Image:       "https://somafm.com/img3/deepspaceone-400.png",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/deepspaceone130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/deepspaceone.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "DEF CON Radio",
		Description: "Music for Hacking. The DEF CON Year-Round Channel.",
		Image:       "https://somafm.com/img3/defcon400.png",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/defcon130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/defcon256.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Department Store Christmas",
		Description: "Holiday Elevator Music from a more innocent time.",
		Image:       "https://somafm.com/img3/deptstore400.jpg",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/deptstore130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/deptstore256.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Digitalis",
		Description: "Digitally affected analog rock to calm the agitated heart.",
		Image:       "https://somafm.com/img3/digitalis-400.png",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/digitalis130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/digitalis256.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Doomed",
		Description: "Where every day is Halloween: Dark industrial/ambient music for tortured souls.",
		Image:       "https://somafm.com/img3/doomed-400.png",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/doomed130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/doomed256.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Drone Zone",
		Description: "Served best chilled, safe with most medications. Atmospheric textures with minimal beats.",
		Image:       "https://somafm.com/img3/dronezone-400.png",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/dronezone130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/dronezone256.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Dub Step Beyond",
		Description: "Dubstep, Dub and Deep Bass. May damage speakers at high volume.",
		Image:       "https://somafm.com/img3/dubstep-400.png",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/dubstep130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/dubstep256.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Fluid",
		Description: "Drown in the electronic sound of instrumental hiphop, future soul and liquid trap.",
		Image:       "https://somafm.com/img3/fluid-400.jpg",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/fluid130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/fluid.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Folk Forward",
		Description: "Indie Folk, Alt-folk and the occasional folk classics.",
		Image:       "https://somafm.com/img3/folkfwd-400.jpg",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/folkfwd130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/folkfwd.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Groove Salad",
		Description: "A nicely chilled plate of ambient/downtempo beats and grooves.",
		Image:       "https://somafm.com/img3/groovesalad-400.png",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/groovesalad130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/groovesalad256.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Groove Salad Classic",
		Description: "The classic (early 2000s) version of a nicely chilled plate of ambient/downtempo beats and grooves.",
		Image:       "https://somafm.com/img3/gsclassic400.jpg",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/gsclassic130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/gsclassic.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Heavyweight Reggae",
		Description: "Reggae, Ska, Rocksteady classic and deep tracks.",
		Image:       "https://somafm.com/img3/reggae400.png",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/reggae130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/reggae256.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Illinois Street Lounge",
		Description: "Classic bachelor pad, playful exotica and vintage music of tomorrow.",
		Image:       "https://somafm.com/img3/illstreet-400.jpg",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/illstreet130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/illstreet.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Indie Pop Rocks!",
		Description: "New and classic favorite indie pop tracks.",
		Image:       "https://somafm.com/img3/indiepop-400.png",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/indiepop130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/indiepop.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Jolly Ol' Soul",
		Description: "Where we cut right to the soul of the season.",
		Image:       "https://somafm.com/img3/jollysoul-400.png",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/jollysoul130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/jollysoul.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Left Coast 70s",
		Description: "Mellow album rock from the Seventies. Yacht not required.",
		Image:       "https://somafm.com/img3/seventies400.jpg",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/seventies130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/seventies320.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Lush",
		Description: "Sensuous and mellow female vocals, many with an electronic influence.",
		Image:       "https://somafm.com/img3/lush-400.jpg",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/lush130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/lush.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Metal Detector",
		Description: "From black to doom, prog to sludge, thrash to post, stoner to crossover, punk to industrial.",
		Image:       "https://somafm.com/img3/metal-400.png",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/metal130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/metal.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Mission Control",
		Description: "Celebrating NASA and Space Explorers everywhere.",
		Image:       "https://somafm.com/img3/missioncontrol-400.png",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/missioncontrol130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/missioncontrol.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "n5MD Radio",
		Description: "Emotional Experiments in Music: Ambient, modern composition, post-rock, & experimental electronic music",
		Image:       "https://somafm.com/img3/n5md-400.png",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/n5md130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/n5md.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "PopTron",
		Description: "Electropop and indie dance rock with sparkle and pop.",
		Image:       "https://somafm.com/img3/poptron-400.png",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/poptron130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/poptron.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Secret Agent",
		Description: "The soundtrack for your stylish, mysterious, dangerous life. For Spies and PIs too!",
		Image:       "https://somafm.com/img3/secretagent-400.png",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/secretagent130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/secretagent.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Seven Inch Soul",
		Description: "Vintage soul tracks from the original 45 RPM vinyl.",
		Image:       "https://somafm.com/img3/7soul-400.jpg",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/7soul130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/7soul.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "SF 10-33",
		Description: "Ambient music mixed with the sounds of San Francisco public safety radio traffic.",
		Image:       "https://somafm.com/img3/sf1033-400.png",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/sf1033130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/sf1033.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "SF in SF",
		Description: "Author readings and discussions from the science fiction, fantasy, horror, and genre literary fields.",
		Image:       "https://somafm.com/img3/sfinsf-400.jpg",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/sfinsf130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/sfinsf.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "SF Police Scanner",
		Description: "San Francisco Public Safety Scanner Feed",
		Image:       "https://somafm.com/img3/scanner-400.jpg",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/scanner130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/scanner.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "SomaFM Live",
		Description: "Special Live Events and rebroadcasts of past live events",
		Image:       "https://somafm.com/img3/live-400.jpg",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/live130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/live.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "SomaFM Specials",
		Description: "Now featuring Afternoon Jazz, Wavepool, DubX, The Surf Report & More!",
		Image:       "https://somafm.com/img3/specials-400.jpg",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/specials130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/specials.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Sonic Universe",
		Description: "Transcending the world of jazz with eclectic, avant-garde takes on tradition.",
		Image:       "https://somafm.com/img3/sonicuniverse-400.png",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/sonicuniverse130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/sonicuniverse256.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Space Station Soma",
		Description: "Tune in, turn on, space out. Spaced-out ambient and mid-tempo electronica.",
		Image:       "https://somafm.com/img3/spacestation-400.png",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/spacestation130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/spacestation.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Suburbs of Goa",
		Description: "Desi-influenced Asian world beats and beyond.",
		Image:       "https://somafm.com/img3/suburbsofgoa-400.png",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/suburbsofgoa130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/suburbsofgoa.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Synphaera Radio",
		Description: "Featuring the music from an independent record label focused on modern electronic ambient and space music.",
		Image:       "https://somafm.com/img3/synphaera400.jpg",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/synphaera130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/synphaera256.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "The Dark Zone",
		Description: "The darker side of deep ambient. Music for staring into the Abyss.",
		Image:       "https://somafm.com/img/darkzone-400.jpg",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/darkzone130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/darkzone256.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "The In-Sound",
		Description: "60s/70s Hipster Euro Pop where psychedelic melodies meets groovy vibes.",
		Image:       "https://somafm.com/logos/400/insound-400.jpg",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/insound130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/insound256.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "The Trip",
		Description: "Progressive house / trance. Tip top tunes.",
		Image:       "https://somafm.com/img3/thetrip-400.jpg",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/thetrip130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/thetrip.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "ThistleRadio",
		Description: "Exploring music from Celtic roots and branches",
		Image:       "https://somafm.com/img3/thistle-400.jpg",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/thistle130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/thistle.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Tiki Time",
		Description: "Classic Tiki music and Vintage island rhythms to sip cocktails by.",
		Image:       "https://somafm.com/logos/400/tikitime-400.jpg",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/tikitime130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/tikitime256.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Underground 80s",
		Description: "Early 80s UK Synthpop and a bit of New Wave.",
		Image:       "https://somafm.com/img3/u80s-400.png",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/u80s130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/u80s256.pls"},
		},
	},
	{
		Creator:     "SomaFM",
		Title:       "Vaporwaves",
		Description: "All Vaporwave. All the time.",
		Image:       "https://somafm.com/img3/vaporwaves400.png",
		Source: []ContentDescription{
			{ContentType: "audio/aac", URL: "https://somafm.com/vaporwaves130.pls"},
			{ContentType: "audio/mpeg", URL: "https://somafm.com/vaporwaves.pls"},
		},
	}}
