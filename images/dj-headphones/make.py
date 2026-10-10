import sys
K="#1d1d1b"
def side(cx, cy, s):
    def e(dx, rx, ry, **kw):
        a=" ".join(f'{k.replace("_","-")}="{v}"' for k,v in kw.items())
        return f'<ellipse cx="{cx+dx*s:.1f}" cy="{cy}" rx="{rx*s:.1f}" ry="{ry*s:.1f}" {a}/>'
    sw=14
    out=[]
    out.append(f'<rect x="{cx-88*s:.1f}" y="{cy-190*s:.1f}" width="{56*s:.1f}" height="{90*s:.1f}" rx="14" fill="#9aa3ad" stroke="{K}" stroke-width="{sw}"/>')
    for dy in (165,142):
        out.append(f'<line x1="{cx-76*s:.1f}" y1="{cy-dy*s:.1f}" x2="{cx-44*s:.1f}" y2="{cy-dy*s:.1f}" stroke="{K}" stroke-width="8" stroke-linecap="round"/>')
    out.append(e(65,48,122,fill="#2b2e33",stroke=K,stroke_width=sw))
    out.append(e(0,88,140,fill="#5b6472",stroke=K,stroke_width=sw))
    out.append(f'<path d="M {cx-50*s:.1f} {cy-90*s:.1f} Q {cx-30*s:.1f} {cy-120*s:.1f} {cx+10*s:.1f} {cy-125*s:.1f}" fill="none" stroke="#fff" stroke-opacity="0.35" stroke-width="12" stroke-linecap="round"/>')
    out.append(e(-12,56,98,fill="#00add8",stroke=K,stroke_width=12))
    out.append(e(-12,40,72,fill=K))
    out.append(e(-12,30,56,fill="none",stroke="#4a4f57",stroke_width=4))
    out.append(e(-12,20,38,fill="none",stroke="#4a4f57",stroke_width=4))
    out.append(e(-12,12,20,fill="#f5a623"))
    out.append(e(-12,3,5,fill=K))
    return "\n".join(out)

def svg(cx, cy, s, end_y, ctrl_y):
    xl, xr = cx-60*s, 1300-(cx-60*s)
    band=f"M {xl:.1f} {end_y} C {xl:.1f} {ctrl_y}, {xr:.1f} {ctrl_y}, {xr:.1f} {end_y}"
    peak=0.25*end_y+0.75*ctrl_y
    stripe_off=26
    parts=[f'<path d="{band}" fill="none" stroke="{K}" stroke-width="80" stroke-linecap="round"/>',
           f'<path d="{band}" fill="none" stroke="#5b6472" stroke-width="52" stroke-linecap="round"/>',
           f'<path d="{band}" fill="none" stroke="#00add8" stroke-width="10" stroke-linecap="round" stroke-dasharray="0 140 100000" transform="translate(0 -8)"/>',
           f'<path d="M 470 {peak-12+18:.1f} Q 560 {peak-12:.1f} 650 {peak-12:.1f}" fill="none" stroke="#fff" stroke-opacity="0.3" stroke-width="10" stroke-linecap="round"/>',
           side(cx, cy, s),
           f'<g transform="translate(1300 0) scale(-1 1)">{side(cx, cy, s)}</g>']
    return '<svg xmlns="http://www.w3.org/2000/svg" width="1300" height="1392" viewBox="0 0 1300 1392">\n'+"\n".join(parts)+"\n</svg>\n"

variants={
  "dj_headphones": dict(cx=140, cy=710, s=1.0, end_y=600, ctrl_y=200),
  "dj_headphones_afro": dict(cx=110, cy=640, s=1.1, end_y=500, ctrl_y=-60),
}
for name,p in variants.items():
    open(f"{sys.argv[1]}/{name}.svg","w").write(svg(**p))
