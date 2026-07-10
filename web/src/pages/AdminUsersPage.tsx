import { useEffect, useState, type FormEvent } from "react";
import * as usersApi from "../api/users";
import * as rolesApi from "../api/roles";
import { useApiAction } from "../hooks/useApiAction";
import { RequestResult } from "../components/RequestResult";
import { useToast } from "../context/ToastContext";

const EMPTY_CREATE: usersApi.CreateUserRequest = {
  email: "",
  username: "",
  password: "",
  firstName: "",
  lastName: "",
  roleId: "",
};

export function AdminUsersPage() {
  const listAction = useApiAction(usersApi.listUsers);
  const createAction = useApiAction(usersApi.createUser);
  const deleteAction = useApiAction(usersApi.deleteUser);
  const blockAction = useApiAction(usersApi.blockUser);
  const unblockAction = useApiAction(usersApi.unblockUser);
  const rolesAction = useApiAction(rolesApi.listRoles);
  const toast = useToast();

  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [createForm, setCreateForm] = useState(EMPTY_CREATE);
  const [blockReason, setBlockReason] = useState<Record<string, string>>({});

  const refresh = () => void listAction.run({ page, perPage: 10, search: search || undefined });

  useEffect(() => {
    refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page]);

  useEffect(() => {
    void rolesAction.run();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const onSearch = (e: FormEvent) => {
    e.preventDefault();
    setPage(1);
    void listAction.run({ page: 1, perPage: 10, search: search || undefined });
  };

  const onCreate = async (e: FormEvent) => {
    e.preventDefault();
    try {
      await createAction.run(createForm);
      toast.success("User created");
      setCreateForm(EMPTY_CREATE);
      refresh();
    } catch {
      /* surfaced below */
    }
  };

  return (
    <div className="flex max-w-4xl flex-col gap-6">
      <div>
        <h2 className="text-lg font-semibold">Admin — Users</h2>
        <p className="text-sm text-slate-500">/v1/users (admin role required)</p>
      </div>

      <div className="card flex flex-col gap-3">
        <form onSubmit={onSearch} className="flex gap-2">
          <input className="input" placeholder="Search…" value={search} onChange={(e) => setSearch(e.target.value)} />
          <button className="btn-secondary" type="submit">
            Search
          </button>
        </form>
        <RequestResult loading={listAction.loading} error={listAction.error} result={null} />
        {listAction.result && listAction.result.items.length > 0 ? (
          <>
            <table className="table-base">
              <thead>
                <tr>
                  <th>Email</th>
                  <th>Username</th>
                  <th>Status</th>
                  <th>2FA</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {listAction.result.items.map((u) => (
                  <tr key={u.id}>
                    <td>{u.email}</td>
                    <td>{u.username}</td>
                    <td>{u.status}</td>
                    <td>{u.twoFAEnabled ? "yes" : "no"}</td>
                    <td className="flex flex-wrap gap-1">
                      {u.status !== "blocked" ? (
                        <>
                          <input
                            className="input !w-32 !py-1 text-xs"
                            placeholder="reason"
                            value={blockReason[u.id] ?? ""}
                            onChange={(e) => setBlockReason((r) => ({ ...r, [u.id]: e.target.value }))}
                          />
                          <button
                            className="btn-secondary !px-2 !py-1 text-xs"
                            onClick={() =>
                              blockAction
                                .run(u.id, blockReason[u.id] || "blocked via test console")
                                .then(refresh)
                                .catch(() => undefined)
                            }
                          >
                            Block
                          </button>
                        </>
                      ) : (
                        <button
                          className="btn-secondary !px-2 !py-1 text-xs"
                          onClick={() => unblockAction.run(u.id).then(refresh).catch(() => undefined)}
                        >
                          Unblock
                        </button>
                      )}
                      <button
                        className="btn-danger !px-2 !py-1 text-xs"
                        onClick={() => deleteAction.run(u.id).then(refresh).catch(() => undefined)}
                      >
                        Delete
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            <div className="flex items-center gap-3 text-xs text-slate-500">
              <button className="btn-secondary !px-2 !py-1" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
                Prev
              </button>
              <span>
                Page {listAction.result.meta?.page} / {listAction.result.meta?.totalPage} (total{" "}
                {listAction.result.meta?.total})
              </span>
              <button
                className="btn-secondary !px-2 !py-1"
                disabled={!listAction.result.meta?.hasNext}
                onClick={() => setPage((p) => p + 1)}
              >
                Next
              </button>
            </div>
          </>
        ) : (
          !listAction.loading && <p className="text-sm text-slate-500">No users found.</p>
        )}
        <RequestResult
          loading={false}
          error={deleteAction.error ?? blockAction.error ?? unblockAction.error}
          result={null}
        />
      </div>

      <form onSubmit={onCreate} className="card flex flex-col gap-3">
        <h3 className="text-sm font-semibold">Create user — POST /v1/users</h3>
        <div>
          <label className="label">Email</label>
          <input
            className="input"
            type="email"
            required
            value={createForm.email}
            onChange={(e) => setCreateForm((f) => ({ ...f, email: e.target.value }))}
          />
        </div>
        <div>
          <label className="label">Username</label>
          <input
            className="input"
            required
            minLength={3}
            value={createForm.username}
            onChange={(e) => setCreateForm((f) => ({ ...f, username: e.target.value }))}
          />
        </div>
        <div>
          <label className="label">Password</label>
          <input
            className="input"
            type="password"
            required
            minLength={8}
            value={createForm.password}
            onChange={(e) => setCreateForm((f) => ({ ...f, password: e.target.value }))}
          />
        </div>
        <div>
          <label className="label">First name</label>
          <input
            className="input"
            required
            value={createForm.firstName}
            onChange={(e) => setCreateForm((f) => ({ ...f, firstName: e.target.value }))}
          />
        </div>
        <div>
          <label className="label">Last name</label>
          <input
            className="input"
            required
            value={createForm.lastName}
            onChange={(e) => setCreateForm((f) => ({ ...f, lastName: e.target.value }))}
          />
        </div>
        <div>
          <label className="label">Role</label>
          <select
            className="input"
            required
            value={createForm.roleId}
            onChange={(e) => setCreateForm((f) => ({ ...f, roleId: e.target.value }))}
            disabled={rolesAction.loading}
          >
            <option value="" disabled>
              {rolesAction.loading ? "Loading roles…" : "Select a role…"}
            </option>
            {rolesAction.result?.map((r) => (
              <option key={r.id} value={r.id}>
                {r.name} ({r.slug})
              </option>
            ))}
          </select>
          <RequestResult loading={false} error={rolesAction.error} result={null} />
        </div>
        <button className="btn self-start" type="submit" disabled={createAction.loading}>
          Create
        </button>
        <RequestResult loading={false} error={createAction.error} result={createAction.result} />
      </form>
    </div>
  );
}
