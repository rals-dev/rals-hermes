// Hair styles per view (docs/decisions/024). The side view faces right;
// compose.ts mirrors whole frames for facing left.

import type { HairStyle } from '../personas'
import type { View } from './body'
import type { Layer } from './layer'

export const HAIR: Readonly<Record<HairStyle, Readonly<Record<View, Layer>>>> = {
  short: {
    front: {
      top: 1,
      rows: [
        '....hhhhhhhh....', // 1
        '...hhHHhhhhhh...', // 2
        '...hhhhhhhhhh...', // 3
        '...hh......hh...', // 4
        '...h........h...', // 5
        '...h........h...', // 6
        '...h........h...', // 7
      ],
    },
    back: {
      top: 1,
      rows: [
        '....hhhhhhhh....', // 1
        '...hhhHHhhhhh...', // 2
        '...hhhhhhhhhh...', // 3
        '...hhhhhhhhhh...', // 4
        '...hhhhhhhhhh...', // 5
        '...hhhhhhhhhh...', // 6
        '...hhhhhhhhhh...', // 7
        '....hhhhhhhh....', // 8
        '.....hhhhhh.....', // 9
      ],
    },
    side: {
      top: 1,
      rows: [
        '.....hhhhhh.....', // 1
        '....hhHHhhhh....', // 2
        '....hhhhhhhh....', // 3
        '....hhhhh.......', // 4
        '....hhh.........', // 5
        '....hh..........', // 6
        '....h...........', // 7
      ],
    },
  },
  messy: {
    front: {
      top: 0,
      rows: [
        '.....h..h.h.....', // 0
        '....hhhhhhhh....', // 1
        '...hhHHhhhhhh...', // 2
        '...hhhhhhhhhh...', // 3
        '...hh......hh...', // 4
        '...h........h...', // 5
        '...h........h...', // 6
        '...h........h...', // 7
      ],
    },
    back: {
      top: 0,
      rows: [
        '.....h..h.h.....', // 0
        '....hhhhhhhh....', // 1
        '...hhhHHhhhhh...', // 2
        '...hhhhhhhhhh...', // 3
        '...hhhhhhhhhh...', // 4
        '...hhhhhhhhhh...', // 5
        '...hhhhhhhhhh...', // 6
        '...hhhhhhhhhh...', // 7
        '....hhhhhhhh....', // 8
        '.....hhhhhh.....', // 9
      ],
    },
    side: {
      top: 0,
      rows: [
        '......h..h......', // 0
        '.....hhhhhh.....', // 1
        '....hhHHhhhh....', // 2
        '....hhhhhhhh....', // 3
        '....hhhhh.......', // 4
        '....hhh.........', // 5
        '....hh..........', // 6
        '....h...........', // 7
      ],
    },
  },
  ponytail: {
    front: {
      top: 1,
      rows: [
        '....hhhhhhhh....', // 1
        '...hhHHhhhhhh...', // 2
        '...hhhhhhhhhh...', // 3
        '...hh......hh...', // 4
        '...h........h...', // 5
        '...h........hho.', // 6
        '...h........hhho', // 7
        '.............oho', // 8
        '..............ho', // 9
        '..............o.', // 10
      ],
    },
    back: {
      top: 1,
      rows: [
        '....hhhhhhhh....', // 1
        '...hhhHHhhhhh...', // 2
        '...hhhhhhhhhh...', // 3
        '...hhhhhhhhhh...', // 4
        '...hhhhhhhhhh...', // 5
        '...hhhhhhhhhh...', // 6
        '...hhhhhhhhhh...', // 7
        '....hhhhhhhh....', // 8
        '.....hhhhhh.....', // 9
        '......ohho......', // 10
        '......ohho......', // 11
        '......oHho......', // 12
        '.......oo.......', // 13
      ],
    },
    side: {
      top: 1,
      rows: [
        '.....hhhhhh.....', // 1
        '....hhHHhhhh....', // 2
        '....hhhhhhhh....', // 3
        '....hhhhh.......', // 4
        '..ohhhh.........', // 5
        '.ohhhh..........', // 6
        '.oh.h...........', // 7
        '.oh.............', // 8
        '..o.............', // 9
      ],
    },
  },
}
