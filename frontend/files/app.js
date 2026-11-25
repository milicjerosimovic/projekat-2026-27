
const API_URL = "/api/files";

let currentFolderId = null;
let folderHistory = [];

function getOwnerId() {
    return document.getElementById("ownerId").value.trim() || localStorage.getItem("jwtUserId") || "";
}

function showStatus(message) {
    document.getElementById("status").textContent = message;
}

async function request(path, options = {}) {
    const token = document.getElementById("token").value.trim() || localStorage.getItem("jwt") || "";

    const headers = new Headers(options.headers || {});

    if (token) {
        headers.set("Authorization", "Bearer " + token);
    }

    const response = await fetch(API_URL + path, {
        ...options,
        headers
    });

    if (!response.ok) {
        throw new Error(await response.text());
    }

    return response;
}
async function getContents(parentId) {
    const params = new URLSearchParams({
        ownerId: getOwnerId()
    });

    if (parentId) {
        params.set("parentId", parentId);
    }

    const response = await request(`/folders?${params}`);
    return await response.json();
}

function updateBreadcrumb() {
    const names = ["Početni folder"];

    for (const folder of folderHistory) {
        if (folder.id) {
            names.push(folder.name);
        }
    }

    if (currentFolderId) {
        names.push(
            document.getElementById("currentFolder").textContent
        );
    }

    document.getElementById("breadcrumb").textContent =
        names.join(" / ");
}

function openRoot() {
    currentFolderId = null;
    folderHistory = [];

    document.getElementById("currentFolder").textContent =
        "Početni folder";

    loadContents();
}

function openFolder(id, name) {
    folderHistory.push({
        id: currentFolderId,
        name: document.getElementById("currentFolder").textContent
    });

    currentFolderId = id;

    document.getElementById("currentFolder").textContent = name;

    loadContents();
}

function goBack() {
    if (folderHistory.length === 0) {
        openRoot();
        return;
    }

    const previous = folderHistory.pop();

    currentFolderId = previous.id;

    document.getElementById("currentFolder").textContent =
        previous.name;

    loadContents();
}

function createButton(text, action) {
    const button = document.createElement("button");

    button.textContent = text;
    button.onclick = action;

    return button;
}

async function buildTree(parentId, container, path = []) {
    const data = await getContents(parentId);

    for (const folder of data.folders || []) {
        const item = document.createElement("div");

        item.className = "tree-item";
        item.textContent = "📁 " + folder.name;

        if (folder.id === currentFolderId) {
            item.classList.add("active");
        }

        item.onclick = () => {
            currentFolderId = folder.id;

            folderHistory = [
                { id: null, name: "Početni folder" },
                ...path
            ];

            document.getElementById("currentFolder").textContent =
                folder.name;

            loadContents();
        };

        container.appendChild(item);

        const children = document.createElement("div");
        children.className = "tree-children";

        container.appendChild(children);

        await buildTree(folder.id, children, [
            ...path,
            { id: folder.id, name: folder.name }
        ]);
    }
}

async function loadTree() {
    const tree = document.getElementById("folderTree");

    tree.innerHTML = "";

    const root = document.createElement("div");

    root.className = "tree-item";
    root.textContent = "📁 Početni folder";
    root.onclick = openRoot;

    if (!currentFolderId) {
        root.classList.add("active");
    }

    tree.appendChild(root);

    const children = document.createElement("div");
    children.className = "tree-children";

    tree.appendChild(children);

    await buildTree(null, children);
}

async function loadContents() {
    try {
        const data = await getContents(currentFolderId);

        const foldersList = document.getElementById("folders");
        const filesList = document.getElementById("files");

        foldersList.innerHTML = "";
        filesList.innerHTML = "";

        for (const folder of data.folders || []) {
            const li = document.createElement("li");

            li.textContent = "📁 " + folder.name;

            li.append(
                createButton("Otvori", () =>
                    openFolder(folder.id, folder.name)
                ),
                createButton("Preimenuj", () =>
                    renameItem("folders", folder.id)
                )
            );

            foldersList.appendChild(li);
        }

        for (const file of data.files || []) {
            const li = document.createElement("li");

            li.textContent = "📄 " + file.name;

            li.append(
                createButton("Preimenuj", () =>
                    renameItem("files", file.id)
                ),
                createButton("Nova verzija", () =>
                    uploadVersion(file.id)
                )
            );

            filesList.appendChild(li);
        }

        updateBreadcrumb();
        await loadTree();

        showStatus("Sadržaj je učitan.");

    } catch (error) {
        showStatus(error.message);
    }
}

async function createFolder() {
    const name = document.getElementById("folderName").value.trim();

    if (!name) {
        showStatus("Unesite naziv foldera.");
        return;
    }

    const body = {
        name: name,
        ownerId: getOwnerId()
    };

    if (currentFolderId) {
        body.parentId = currentFolderId;
    }

    try {
        await request("/folders", {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify(body)
        });

        document.getElementById("folderName").value = "";

        await loadContents();

        showStatus("Folder je uspešno kreiran.");

    } catch (error) {
        showStatus(error.message);
    }
}

async function renameItem(type, id) {
    const name = prompt("Unesite novi naziv:");

    if (!name || !name.trim()) return;

    try {
        await request(`/${type}/${id}`, {
            method: "PATCH",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify({
                ownerId: getOwnerId(),
                name: name.trim()
            })
        });

        await loadContents();

        showStatus("Naziv je uspešno promenjen.");

    } catch (error) {
        showStatus(error.message);
    }
}

async function uploadFile() {
    if (!currentFolderId) {
        showStatus("Prvo otvorite folder.");
        return;
    }

    const input = document.getElementById("fileInput");
    const file = input.files[0];

    if (!file) {
        showStatus("Izaberite datoteku.");
        return;
    }

    const form = new FormData();

    form.append("ownerId", getOwnerId());
    form.append("parentId", currentFolderId);
    form.append("file", file);

    try {
        await request("/files", {
            method: "POST",
            body: form
        });

        input.value = "";

        await loadContents();

        showStatus("Datoteka je uspešno otpremljena.");

    } catch (error) {
        showStatus(error.message);
    }
}

function uploadVersion(id) {
    const input = document.createElement("input");

    input.type = "file";

    input.onchange = async () => {
        const file = input.files[0];

        if (!file) return;

        const form = new FormData();

        form.append("ownerId", getOwnerId());
        form.append("file", file);

        try {
            await request(`/files/${id}/versions`, {
                method: "POST",
                body: form
            });

            await loadContents();

            showStatus("Nova verzija je uspešno otpremljena.");

        } catch (error) {
            showStatus(error.message);
        }
    };

    input.click();
}

openRoot();

