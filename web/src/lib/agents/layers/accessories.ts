// Accessories per view (docs/decisions/024), drawn in mid-tone greys so
// they read against the black outline and the charcoal floor. Held items
// (magnifier, clipboard) are hidden from behind.

import type { Accessory } from '../personas'
import type { View } from './body'
import { EMPTY, type Layer } from './layer'

export const ACCESSORIES: Readonly<Record<Accessory, Readonly<Record<View, Layer>>>> = {
  headset: {
    front: {
      top: 1,
      rows: [
        '....KKKKKKKK....', // 1
        '...K............', // 2
        '..K.............', // 3
        'okk.............', // 4
        'okk.............', // 5
        'okk.............', // 6
        'okk.............', // 7
        '...K............', // 8
        '....Kmm.........', // 9
      ],
    },
    back: {
      top: 1,
      rows: [
        '....KKKKKKKK....', // 1
        '............K...', // 2
        '.............K..', // 3
        '.............kko', // 4
        '.............kko', // 5
        '.............kko', // 6
        '.............kko', // 7
      ],
    },
    side: {
      top: 1,
      rows: [
        '.....KKKKKK.....', // 1
        '....K...........', // 2
      ],
    },
  },
  tie: {
    front: {
      top: 12,
      rows: [
        '.......tt.......', // 12
        '.......tt.......', // 13
        '.......tt.......', // 14
        '.......tt.......', // 15
      ],
    },
    back: EMPTY,
    side: {
      top: 12,
      rows: [
        '.........t......', // 12
        '.........t......', // 13
        '.........t......', // 14
      ],
    },
  },
  headphones: {
    front: {
      top: 0,
      rows: [
        '....KKKKKKKK....', // 0
        '...K........K...', // 1
        '..K..........K..', // 2
        'okk..........kko', // 3
        'ock..........kco', // 4
        'ock..........kco', // 5
        'ock..........kco', // 6
        'okk..........kko', // 7
      ],
    },
    back: {
      top: 0,
      rows: [
        '....KKKKKKKK....', // 0
        '...K........K...', // 1
        '..K..........K..', // 2
        'okk..........kko', // 3
        'ock..........kco', // 4
        'ock..........kco', // 5
        'ock..........kco', // 6
        'okk..........kko', // 7
      ],
    },
    side: {
      top: 0,
      rows: [
        '.....KKKKKK.....', // 0
        '.....K..........', // 1
        '.....K..........', // 2
        '.....K..........', // 3
        '....okkko.......', // 4
        '....okcko.......', // 5
        '....okkko.......', // 6
        '.....ooo........', // 7
      ],
    },
  },
  hood: {
    front: {
      top: 10,
      rows: [
        '....oCCCCCCo....', // 10
        '.......ww.......', // 11
      ],
    },
    back: {
      top: 10,
      rows: [
        '...oCCCCCCCCo...', // 10
        '.....CCCCCC.....', // 11
      ],
    },
    side: {
      top: 10,
      rows: [
        '...oCCC.........', // 10
        '.........w......', // 11
      ],
    },
  },
  goggles: {
    front: {
      top: 3,
      rows: [
        '...KKggKKggKK...', // 3
      ],
    },
    back: {
      top: 3,
      rows: [
        '...kkkkkkkkkk...', // 3
      ],
    },
    side: {
      top: 3,
      rows: [
        '....KKKKKggK....', // 3
      ],
    },
  },
  magnifier: {
    front: {
      top: 11,
      rows: [
        '.............kk.', // 11
        '............kggk', // 12
        '............kgwk', // 13
        '.............kk.', // 14
        '............k...', // 15
      ],
    },
    back: EMPTY,
    side: {
      top: 10,
      rows: [
        '...........kk...', // 10
        '..........kggk..', // 11
        '..........kgwk..', // 12
        '...........kk...', // 13
        '..........k.....', // 14
      ],
    },
  },
  clipboard: {
    front: {
      top: 12,
      rows: [
        '.......kk.......', // 12
        '....owwwwwwo....', // 13
        '....owllllwo....', // 14
        '....owwwwwwo....', // 15
        '....owlllwwo....', // 16
        '....owwwwwwo....', // 17
      ],
    },
    back: EMPTY,
    side: {
      top: 11,
      rows: [
        '...........k....', // 11
        '...........wo...', // 12
        '...........wo...', // 13
        '...........wo...', // 14
        '...........wo...', // 15
        '...........wo...', // 16
      ],
    },
  },
  glasses: {
    front: {
      top: 5,
      rows: [
        '....KKK..KKK....', // 5
        '....K.KKKK.K....', // 6
      ],
    },
    back: EMPTY,
    side: {
      top: 4,
      rows: [
        '.........KKK....', // 4
        '........KK.K....', // 5
      ],
    },
  },
}
