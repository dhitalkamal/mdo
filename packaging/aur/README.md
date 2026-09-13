# AUR packaging

`PKGBUILD` for the `mdo-bin` AUR package (installs the prebuilt release binary).

Publishing to the AUR needs an AUR account with an SSH key registered (this can
only be done by the maintainer, not in CI). Steps:

```sh
git clone ssh://aur@aur.archlinux.org/mdo-bin.git
cp PKGBUILD mdo-bin/
cd mdo-bin
makepkg --printsrcinfo > .SRCINFO
git add PKGBUILD .SRCINFO
git commit -m "mdo-bin 0.1.0"
git push
```

On each new release, bump `pkgver` in `PKGBUILD`, refresh the `sha256sums_*`
(from the release tarballs), regenerate `.SRCINFO`, and push.
