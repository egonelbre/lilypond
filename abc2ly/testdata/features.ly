\version "2.24.1"
\header { tagline = #f }

\score {
  \header {
      piece = "Features"
      composer = "Composer"
      history = "12 märts 1981"
  }
  \new Staff{
  \accidentalStyle modern
    \time 3/4 \key c \major
    a'4. b'8 a'4 b'2. | a'8 b'8 c'8 d'8 f'16 e'8 f'16 | \break
    a'4 ^"C" b'4 d'4 ^"D" | \break
    <fis' e' des'>2.~ | <fis' e' des'>2. | \break
    a'4. r8 b'4 | bes'2.~ | \break
    bes'2. | b'2. | \break
    aes'4 a'4 aes'4 | cis'4 c'4 cis'4 | \break
    \set Score.repeatCommands = #'(start-repeat) b'4 a'4 b'4 \bar "||" b'4 a'4 d'4 \set Score.repeatCommands = #'(end-repeat) \break
    a'4-. b'4-. c'4-. | e'4-^ f'4-. g'4-. | \break
    aes'8 bes'8 aes'8 bes'8 aes'8 bes'8~ | bes'8 a'8 a'4 c'4 | \break
    \key f \major bes'4 a'4 g'4 | bes'4 a'4 g'4 | bis'4 g'4 a'4 |
  }
}
